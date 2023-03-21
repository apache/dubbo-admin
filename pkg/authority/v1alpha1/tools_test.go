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

package v1alpha1_test

import (
	"context"
	"net/url"
	"testing"

	"google.golang.org/grpc/credentials"

	"github.com/apache/dubbo-admin/pkg/authority/jwt"

	"github.com/apache/dubbo-admin/pkg/authority/cert"
	"github.com/apache/dubbo-admin/pkg/authority/config"
	"github.com/apache/dubbo-admin/pkg/authority/k8s"
	"github.com/apache/dubbo-admin/pkg/authority/rule"
	"google.golang.org/grpc/metadata"

	"github.com/apache/dubbo-admin/pkg/authority/v1alpha1"
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
	k8s.Client
}

func (c fakeKubeClient) VerifyServiceAccount(token string, authorizationType string) (*rule.Endpoint, bool) {
	if token == "kubernetes-token" && authorizationType == "kubernetes" {
		return &rule.Endpoint{
			ID: "kubernetes",
		}, true
	}
	if token == "dubbo-token" && authorizationType == "dubbo-ca-token" {
		return &rule.Endpoint{
			ID: "dubbo-endpoint",
		}, true
	}
	return nil, false
}

func TestKubernetes(t *testing.T) {
	t.Parallel()
	_, err := v1alpha1.ExactEndpoint(nil, nil, nil, nil) // nolint: staticcheck
	assert.NotNil(t, err)

	_, err = v1alpha1.ExactEndpoint(context.TODO(), nil, nil, nil)
	assert.NotNil(t, err)

	c := peer.NewContext(context.TODO(), &peer.Peer{Addr: &fakeAddr{}})
	options := &config.Options{
		IsKubernetesConnected: false,
		IsTrustAnyone:         false,
		CertValidity:          24 * 60 * 60 * 1000,
		CaValidity:            365 * 24 * 60 * 60 * 1000,
	}
	storage := cert.NewStorage(options)
	storage.SetAuthorityCert(cert.GenerateAuthorityCert(nil, options.CaValidity))
	storage.AddTrustedCert(storage.GetAuthorityCert())

	// verify failed
	_, err = v1alpha1.ExactEndpoint(c, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)

	options.IsTrustAnyone = true
	// trust anyone
	endpoint, err := v1alpha1.ExactEndpoint(c, storage, options, &fakeKubeClient{})
	assert.Nil(t, err)
	assert.NotNil(t, endpoint)
	assert.Equal(t, "127.0.0.1:12345", endpoint.ID)
	assert.Equal(t, 1, len(endpoint.Ips))

	options.IsTrustAnyone = false

	// empty authorization
	_, err = v1alpha1.ExactEndpoint(c, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)

	// invalid header
	md := metadata.MD{}
	md["authorization"] = []string{"invalid"}
	withAuthorization := metadata.NewIncomingContext(c, md)
	_, err = v1alpha1.ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)

	// invalid token
	md = metadata.MD{}
	md["authorization"] = []string{"Bearer invalid"}
	withAuthorization = metadata.NewIncomingContext(c, md)
	_, err = v1alpha1.ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)

	options.IsKubernetesConnected = true
	options.EnableOIDCCheck = true

	// kubernetes token
	md = metadata.MD{}
	md["authorization"] = []string{"Bearer kubernetes-token"}
	withAuthorization = metadata.NewIncomingContext(c, md)
	endpoint, err = v1alpha1.ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.Nil(t, err)
	assert.NotNil(t, endpoint)
	assert.Equal(t, "kubernetes", endpoint.ID)

	// kubernetes token
	md = metadata.MD{}
	md["authorization"] = []string{"Bearer kubernetes-token"}
	md["authorization-type"] = []string{"kubernetes"}
	withAuthorization = metadata.NewIncomingContext(c, md)
	endpoint, err = v1alpha1.ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.Nil(t, err)
	assert.NotNil(t, endpoint)
	assert.Equal(t, "kubernetes", endpoint.ID)

	// dubbo-ca token
	md = metadata.MD{}
	md["authorization"] = []string{"Bearer dubbo-token"}
	md["authorization-type"] = []string{"dubbo-ca-token"}
	withAuthorization = metadata.NewIncomingContext(c, md)
	endpoint, err = v1alpha1.ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.Nil(t, err)
	assert.NotNil(t, endpoint)
	assert.Equal(t, "dubbo-endpoint", endpoint.ID)
}

func TestJwt(t *testing.T) {
	t.Parallel()

	c := peer.NewContext(context.TODO(), &peer.Peer{Addr: &fakeAddr{}})
	options := &config.Options{
		IsKubernetesConnected: false,
		IsTrustAnyone:         false,
		CertValidity:          24 * 60 * 60 * 1000,
		CaValidity:            365 * 24 * 60 * 60 * 1000,
	}
	storage := cert.NewStorage(options)
	storage.SetAuthorityCert(cert.GenerateAuthorityCert(nil, options.CaValidity))
	storage.AddTrustedCert(storage.GetAuthorityCert())

	options.IsTrustAnyone = false

	// invalid token
	md := metadata.MD{}
	md["authorization"] = []string{"Bearer kubernetes-token"}
	md["authorization-type"] = []string{"dubbo-jwt"}
	withAuthorization := metadata.NewIncomingContext(c, md)
	_, err := v1alpha1.ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)

	// invalid jwt data
	token, err := jwt.NewClaims("123", "123", 60*1000).Sign(storage.GetAuthorityCert().PrivateKey)
	assert.Nil(t, err)

	md["authorization"] = []string{"Bearer " + token}
	md["authorization-type"] = []string{"dubbo-jwt"}
	withAuthorization = metadata.NewIncomingContext(c, md)
	_, err = v1alpha1.ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)

	// dubbo-ca token
	md = metadata.MD{}
	originEndpoint := &rule.Endpoint{
		ID:       "dubbo-endpoint",
		SpiffeID: "spiffe://cluster.local",
		Ips:      []string{"127.0.0.1"},
		KubernetesEnv: &rule.KubernetesEnv{
			Namespace: "default",
		},
	}
	token, err = jwt.NewClaims(originEndpoint.SpiffeID, originEndpoint.ToString(), 60*1000).Sign(storage.GetAuthorityCert().PrivateKey)
	assert.Nil(t, err)

	md["authorization"] = []string{"Bearer " + token}
	md["authorization-type"] = []string{"dubbo-jwt"}
	withAuthorization = metadata.NewIncomingContext(c, md)
	endpoint, err := v1alpha1.ExactEndpoint(withAuthorization, storage, options, &fakeKubeClient{})
	assert.Nil(t, err)
	assert.NotNil(t, endpoint)
	assert.Equal(t, originEndpoint, endpoint)
}

func TestConnection(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		IsTrustAnyone:         false,
		CertValidity:          24 * 60 * 60 * 1000,
		CaValidity:            365 * 24 * 60 * 60 * 1000,
	}
	storage := cert.NewStorage(options)
	storage.SetAuthorityCert(cert.GenerateAuthorityCert(nil, options.CaValidity))
	storage.AddTrustedCert(storage.GetAuthorityCert())

	options.IsTrustAnyone = false

	// invalid token
	c := peer.NewContext(context.TODO(), &peer.Peer{
		Addr:     &fakeAddr{},
		AuthInfo: credentials.TLSInfo{},
	})

	_, err := v1alpha1.ExactEndpoint(c, storage, options, &fakeKubeClient{})
	assert.NotNil(t, err)
	// invalid token
	c = peer.NewContext(context.TODO(), &peer.Peer{
		Addr:     &fakeAddr{},
		AuthInfo: &credentials.TLSInfo{},
	})

	_, err = v1alpha1.ExactEndpoint(c, storage, options, &fakeKubeClient{})
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

	endpoint, err := v1alpha1.ExactEndpoint(c, storage, options, &fakeKubeClient{})
	assert.Nil(t, err)
	assert.NotNil(t, endpoint)
	assert.Equal(t, "127.0.0.1:12345", endpoint.ID)
	assert.Equal(t, "spiffe://cluster.local", endpoint.SpiffeID)
}
