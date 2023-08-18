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

package endpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core/cert/provider"
	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/jwt"

	"google.golang.org/grpc/credentials"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func ExactEndpoint(c context.Context, certStorage *provider.CertStorage, options *dubbo_cp.Config, certClient provider.Client) (*endpoint.Endpoint, error) {
	if c == nil {
		return nil, fmt.Errorf("context is nil")
	}

	p, ok := peer.FromContext(c)
	if !ok {
		return nil, fmt.Errorf("failed to get peer from context")
	}

	endpoints, endpointErr := tryFromHeader(c, certStorage, options, certClient)
	if endpointErr == nil {
		return endpoints, nil
	}

	endpoints, connectionErr := tryFromConnection(p)
	if connectionErr == nil {
		return endpoints, nil
	}

	if !options.Security.IsTrustAnyone && connectionErr != nil {
		return nil, fmt.Errorf("failed to get endpoint from header: %s. Failed to get endpoint from storage: %s. RemoteAddr: %s",
			endpointErr.Error(), connectionErr.Error(), p.Addr.String())
	}

	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return nil, err
	}

	return &endpoint.Endpoint{
		ID:  p.Addr.String(),
		Ips: []string{host},
	}, nil
}

func tryFromHeader(c context.Context, certStorage *provider.CertStorage, options *dubbo_cp.Config, kubeClient provider.Client) (*endpoint.Endpoint, error) {
	// TODO refactor as coreos/go-oidc
	authorization := metadata.ValueFromIncomingContext(c, "authorization")
	if len(authorization) != 1 {
		return nil, fmt.Errorf("failed to get Authorization header from context")
	}

	if !strings.HasPrefix(authorization[0], "Bearer ") {
		return nil, fmt.Errorf("failed to get Authorization header from context")
	}

	token := strings.ReplaceAll(authorization[0], "Bearer ", "")

	authorizationTypes := metadata.ValueFromIncomingContext(c, "authorization-type")
	authorizationType := "kubernetes"

	if len(authorizationTypes) == 1 {
		authorizationType = authorizationTypes[0]
	}

	if authorizationType == "dubbo-jwt" {
		for _, c := range certStorage.GetTrustedCerts() {
			claims, err := jwt.Verify(&c.PrivateKey.PublicKey, token)
			if err != nil {
				continue
			}
			endpoint := &endpoint.Endpoint{SpiffeID: claims.Subject}
			err = json.Unmarshal([]byte(claims.Extensions), endpoint)
			if err != nil {
				continue
			}
			return endpoint, nil
		}
		return nil, fmt.Errorf("failed to verify Authorization header from dubbo-jwt")
	}

	if options.KubeConfig.IsKubernetesConnected && options.Security.EnableOIDCCheck {
		endpoint, ok := kubeClient.VerifyServiceAccount(token, authorizationType)
		if !ok {
			return nil, fmt.Errorf("failed to verify Authorization header from kubernetes")
		}
		return endpoint, nil
	}

	return nil, fmt.Errorf("failed to verify Authorization header")
}

func tryFromConnection(p *peer.Peer) (*endpoint.Endpoint, error) {
	if p.AuthInfo != nil && p.AuthInfo.AuthType() == "tls" {
		tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
		if !ok {
			return nil, fmt.Errorf("failed to get TLSInfo from peer")
		}

		host, _, err := net.SplitHostPort(p.Addr.String())
		if err != nil {
			return nil, err
		}
		if tlsInfo.SPIFFEID == nil {
			return nil, fmt.Errorf("failed to get SPIFFE ID from peer")
		}
		return &endpoint.Endpoint{
			ID:       p.Addr.String(),
			SpiffeID: tlsInfo.SPIFFEID.String(),
			Ips:      []string{host},
		}, nil
	}
	return nil, fmt.Errorf("failed to get TLSInfo from peer")
}
