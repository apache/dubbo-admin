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
	"net/url"
	"testing"

	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/config/kube"
	"github.com/apache/dubbo-admin/pkg/config/security"
	"github.com/apache/dubbo-admin/pkg/core/cert/provider"
	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/jwt"

	"google.golang.org/grpc/credentials"

	"google.golang.org/grpc/metadata"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/peer"
)

type fakeAddr struct{}

func (f *fakeAddr) String() string {
	return "127.0.0.1:12345"
}

func (f *fakeAddr) Network() string {
	return ""
}

type fakeKubeClient struct {
	provider.Client
}

func (c fakeKubeClient) VerifyServiceAccount(token string, authorizationType string) (*endpoint.Endpoint, bool) {
	if token == "kubernetes-token" && authorizationType == "kubernetes" {
		return &endpoint.Endpoint{
			ID: "kubernetes",
		}, true
	}
	if token == "dubbo-token" && authorizationType == "dubbo-ca-token" {
		return &endpoint.Endpoint{
			ID: "dubbo-endpoint",
		}, true
	}
	return nil, false
}

func TestKubernetes(t *testing.T) {
	t.Parallel()
	_, err := ExactEndpoint(nil, nil, nil, nil) // nolint: staticcheck
	assert.NotNil(t, err)

	_, err = ExactEndpoint(context.TODO(), nil, nil, nil)
	assert.NotNil(t, err)

	c := peer.NewContext(context.TODO(), &peer.Peer{Addr: &fakeAddr{}})
	options := &dubbo_cp.Config{
		Security: security.SecurityConfig{
			IsTrustAnyone: false,
			CertValidity:  24 * 60 * 60 * 1000,
			CaValidity:    365 * 24 * 60 * 60 * 1000,
		},
		KubeConfig: kube.KubeConfig{
			IsKubernetesConnected: false,
		},
	}
	storage := provider.NewStorage(options, &provider.ClientImpl{})
	storage.SetAuthorityCert(provider.GenerateAuthorityCert(nil, options.Security.CaValidity))
	storage.AddTrustedCert(storage.GetAuthorityCert())

	// verify failed
	_, err = ExactEndpoint(c, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)

	options.Security.IsTrustAnyone = true
	// trust anyone
	endpoint, err := ExactEndpoint(c, storage, options, &fakeKubeClient{})
	assert.Nil(t, err)
	assert.NotNil(t, endpoint)
	assert.Equal(t, "127.0.0.1:12345", endpoint.ID)
	assert.Equal(t, 1, len(endpoint.Ips))

	options.Security.IsTrustAnyone = false

	// empty authorization
	_, err = ExactEndpoint(c, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)

	// invalid header
	md := metadata.MD{}
	md["authorization"] = []string{"invalid"}
	withAuthorization := metadata.NewIncomingContext(c, md)
	_, err = ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)

	// invalid token
	md = metadata.MD{}
	md["authorization"] = []string{"Bearer invalid"}
	withAuthorization = metadata.NewIncomingContext(c, md)
	_, err = ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)

	options.KubeConfig.IsKubernetesConnected = true
	options.Security.EnableOIDCCheck = true

	// kubernetes token
	md = metadata.MD{}
	md["authorization"] = []string{"Bearer kubernetes-token"}
	withAuthorization = metadata.NewIncomingContext(c, md)
	endpoint, err = ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.Nil(t, err)
	assert.NotNil(t, endpoint)
	assert.Equal(t, "kubernetes", endpoint.ID)

	// kubernetes token
	md = metadata.MD{}
	md["authorization"] = []string{"Bearer kubernetes-token"}
	md["authorization-type"] = []string{"kubernetes"}
	withAuthorization = metadata.NewIncomingContext(c, md)
	endpoint, err = ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.Nil(t, err)
	assert.NotNil(t, endpoint)
	assert.Equal(t, "kubernetes", endpoint.ID)

	// dubbo-ca token
	md = metadata.MD{}
	md["authorization"] = []string{"Bearer dubbo-token"}
	md["authorization-type"] = []string{"dubbo-ca-token"}
	withAuthorization = metadata.NewIncomingContext(c, md)
	endpoint, err = ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.Nil(t, err)
	assert.NotNil(t, endpoint)
	assert.Equal(t, "dubbo-endpoint", endpoint.ID)
}

func TestJwt(t *testing.T) {
	t.Parallel()

	c := peer.NewContext(context.TODO(), &peer.Peer{Addr: &fakeAddr{}})
	options := &dubbo_cp.Config{
		Security: security.SecurityConfig{
			IsTrustAnyone: false,
			CertValidity:  24 * 60 * 60 * 1000,
			CaValidity:    365 * 24 * 60 * 60 * 1000,
		},
		KubeConfig: kube.KubeConfig{
			IsKubernetesConnected: false,
		},
	}
	storage := provider.NewStorage(options, &provider.ClientImpl{})
	storage.SetAuthorityCert(provider.GenerateAuthorityCert(nil, options.Security.CaValidity))
	storage.AddTrustedCert(storage.GetAuthorityCert())

	options.Security.IsTrustAnyone = false

	// invalid token
	md := metadata.MD{}
	md["authorization"] = []string{"Bearer kubernetes-token"}
	md["authorization-type"] = []string{"dubbo-jwt"}
	withAuthorization := metadata.NewIncomingContext(c, md)
	_, err := ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)

	// invalid jwt data
	token, err := jwt.NewClaims("123", "123", "test", 60*1000).Sign(storage.GetAuthorityCert().PrivateKey)
	assert.Nil(t, err)

	md["authorization"] = []string{"Bearer " + token}
	md["authorization-type"] = []string{"dubbo-jwt"}
	withAuthorization = metadata.NewIncomingContext(c, md)
	_, err = ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)

	// dubbo-ca token
	md = metadata.MD{}
	originEndpoint := &endpoint.Endpoint{
		ID:       "dubbo-endpoint",
		SpiffeID: "spiffe://cluster.local",
		Ips:      []string{"127.0.0.1"},
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "default",
		},
	}
	token, err = jwt.NewClaims(originEndpoint.SpiffeID, originEndpoint.ToString(), "test", 60*1000).Sign(storage.GetAuthorityCert().PrivateKey)
	assert.Nil(t, err)

	md["authorization"] = []string{"Bearer " + token}
	md["authorization-type"] = []string{"dubbo-jwt"}
	withAuthorization = metadata.NewIncomingContext(c, md)
	endpoint, err := ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.Nil(t, err)
	assert.NotNil(t, endpoint)
	assert.Equal(t, originEndpoint, endpoint)
}

func TestConnection(t *testing.T) {
	t.Parallel()

	options := &dubbo_cp.Config{
		Security: security.SecurityConfig{
			IsTrustAnyone: false,
			CertValidity:  24 * 60 * 60 * 1000,
			CaValidity:    365 * 24 * 60 * 60 * 1000,
		},
		KubeConfig: kube.KubeConfig{
			IsKubernetesConnected: false,
		},
	}
	storage := provider.NewStorage(options, &provider.ClientImpl{})
	storage.SetAuthorityCert(provider.GenerateAuthorityCert(nil, options.Security.CaValidity))
	storage.AddTrustedCert(storage.GetAuthorityCert())

	options.Security.IsTrustAnyone = false

	// invalid token
	c := peer.NewContext(context.TODO(), &peer.Peer{
		Addr:     &fakeAddr{},
		AuthInfo: credentials.TLSInfo{},
	})

	_, err := ExactEndpoint(c, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)
	// invalid token
	c = peer.NewContext(context.TODO(), &peer.Peer{
		Addr:     &fakeAddr{},
		AuthInfo: &credentials.TLSInfo{},
	})

	_, err = ExactEndpoint(c, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)

	// valid token
	c = peer.NewContext(context.TODO(), &peer.Peer{
		Addr: &fakeAddr{},
		AuthInfo: credentials.TLSInfo{
			SPIFFEID: &url.URL{
				Scheme: "spiffe",
				Host:   "cluster.local",
			},
		},
	})

	endpoint, err := ExactEndpoint(c, storage, options, &fakeKubeClient{})
	assert.Nil(t, err)
	assert.NotNil(t, endpoint)
	assert.Equal(t, "127.0.0.1:12345", endpoint.ID)
	assert.Equal(t, "spiffe://cluster.local", endpoint.SpiffeID)
}
