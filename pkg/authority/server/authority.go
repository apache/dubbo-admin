// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"net/http"
	"time"

	"github.com/apache/dubbo-admin/api/ca"
	"github.com/apache/dubbo-admin/pkg/authority/patch"
	"github.com/apache/dubbo-admin/pkg/authority/webhook"
	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	cert "github.com/apache/dubbo-admin/pkg/core/cert/provider"
	"github.com/apache/dubbo-admin/pkg/core/jwt"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/tools/endpoint"
	"google.golang.org/grpc/peer"
)

type AuthorityService struct {
	ca.UnimplementedAuthorityServiceServer
	Options     *dubbo_cp.Config
	CertClient  cert.Client
	CertStorage *cert.CertStorage

	WebhookServer *webhook.Webhook
	JavaInjector  *patch.JavaSdk
}

func (s *AuthorityService) NeedLeaderElection() bool {
	return false
}

func (s *AuthorityService) Start(stop <-chan struct{}) error {
	errChan := make(chan error)
	if s.Options.KubeConfig.InPodEnv {
		go func() {
			err := s.WebhookServer.Server.ListenAndServeTLS("", "")
			if err != nil {
				switch err {
				case http.ErrServerClosed:
					logger.Sugar().Info("[Webhook] shutting down HTTP Server")
				default:
					logger.Sugar().Error(err, "[Webhook] could not start an HTTP Server")
					errChan <- err
				}
			}
		}()
		s.CertClient.UpdateWebhookConfig(s.Options, s.CertStorage)
		select {
		case <-stop:
			logger.Sugar().Info("[Webhook] stopping Authority")
			if s.WebhookServer.Server != nil {
				return s.WebhookServer.Server.Shutdown(context.Background())
			}
		case err := <-errChan:
			return err
		}
	}
	return nil
}

func NewServer(options *dubbo_cp.Config) *AuthorityService {
	return &AuthorityService{
		Options: options,
	}
}

func (s *AuthorityService) CreateIdentity(
	c context.Context,
	req *ca.IdentityRequest,
) (*ca.IdentityResponse, error) {
	if req.Csr == "" {
		return &ca.IdentityResponse{
			Success: false,
			Message: "CSR is empty.",
		}, nil
	}

	csr, err := cert.LoadCSR(req.Csr)
	if csr == nil || err != nil {
		return &ca.IdentityResponse{
			Success: false,
			Message: "Decode csr failed.",
		}, nil
	}

	p, _ := peer.FromContext(c)
	endpoint, err := endpoint.ExactEndpoint(c, s.CertStorage, s.Options, s.CertClient)
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to exact endpoint from context: %v. RemoteAddr: %s", err, p.Addr.String())

		return &ca.IdentityResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	certPem, err := cert.SignFromCSR(csr, endpoint, s.CertStorage.GetAuthorityCert(), s.Options.Security.CertValidity)
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to sign certificate from csr: %v. RemoteAddr: %s", err, p.Addr.String())

		return &ca.IdentityResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	logger.Sugar().Infof("[Authority] Success to sign certificate from csr. RemoteAddr: %s", p.Addr.String())

	token, err := jwt.NewClaims(endpoint.SpiffeID, endpoint.ToString(), endpoint.ID, s.Options.Security.CertValidity).Sign(s.CertStorage.GetAuthorityCert().PrivateKey)
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to sign jwt token: %v. RemoteAddr: %s", err, p.Addr.String())

		return &ca.IdentityResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	var trustedCerts []string
	var trustedTokenPublicKeys []string
	for _, c := range s.CertStorage.GetTrustedCerts() {
		trustedCerts = append(trustedCerts, c.CertPem)
		trustedTokenPublicKeys = append(trustedTokenPublicKeys, cert.EncodePublicKey(&c.PrivateKey.PublicKey))
	}
	return &ca.IdentityResponse{
		Success:                true,
		Message:                "OK",
		CertPem:                certPem,
		TrustCerts:             trustedCerts,
		Token:                  token,
		TrustedTokenPublicKeys: trustedTokenPublicKeys,
		RefreshTime:            time.Now().UnixMilli() + (s.Options.Security.CertValidity / 2),
		ExpireTime:             time.Now().UnixMilli() + s.Options.Security.CertValidity,
	}, nil
}
