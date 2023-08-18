/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package storage

import (
	"testing"

	dubbo_apache_org_v1alpha1 "github.com/apache/dubbo-admin/api/resource/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/model"
	"github.com/apache/dubbo-admin/pkg/core/schema/collection"
	"github.com/apache/dubbo-admin/pkg/core/schema/collections"
	"github.com/apache/dubbo-admin/pkg/core/schema/gvk"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestAuthenticationSelect_Empty(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthenticationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)
	policy.Action = "ALLOW"
	policy.PortLevel = []*dubbo_apache_org_v1alpha1.AuthenticationPolicyPortLevel{
		{
			Port:   8080,
			Action: "DENY",
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthenticationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: pb,
			},
		},
	}

	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	generated, err := origin.Exact(gen, &endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "name/ns", authentication.Key)
			assert.Equal(t, "ALLOW", authentication.Spec.Action)
			assert.Equal(t, 1, len(authentication.Spec.PortLevel))
			assert.Equal(t, "DENY", authentication.Spec.PortLevel[0].Action)
		}
	}
}

func TestAuthenticationSelect_NoSelector(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthenticationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)
	policy.Action = "ALLOW"

	origin := &OriginImpl{
		Gvk: gvk.AuthenticationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: pb,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "test",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "name/ns", authentication.Key)
			assert.Equal(t, "ALLOW", authentication.Spec.Action)
		}
	}
}

func TestAuthenticationSelect_Namespace(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthenticationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)
	policy.Action = "ALLOW"
	policy.Selector = []*dubbo_apache_org_v1alpha1.AuthenticationPolicySelector{
		{
			Namespaces: []string{"test"},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthenticationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: pb,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "test",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authentication.Spec.Action)
		}
	}

	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "demo",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))
}

func TestAuthenticationSelect_EndpointNil(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthenticationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)
	policy.Action = "ALLOW"
	policy.Selector = []*dubbo_apache_org_v1alpha1.AuthenticationPolicySelector{
		{
			Namespaces: []string{"test"},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthenticationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: pb,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	generated, err := origin.Exact(gen, nil)
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authentication.Spec.Action)
		}
	}
}

func TestAuthenticationSelect_NotNamespace(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthenticationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)
	policy.Action = "ALLOW"
	policy.Selector = []*dubbo_apache_org_v1alpha1.AuthenticationPolicySelector{
		{
			NotNamespaces: []string{"test"},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthenticationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: pb,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "test",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "demo",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data1 := generated.Data

	for _, anyMessage := range data1 {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authentication.Spec.Action)
		}
	}
}

func TestAuthenticationSelect_IpBlocks_ErrFmt(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthenticationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)
	policy.Action = "ALLOW"
	policy.Selector = []*dubbo_apache_org_v1alpha1.AuthenticationPolicySelector{
		{
			IpBlocks: []string{"123"},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthenticationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: pb,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127.0.0.2"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127.0.0.3"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data1 := generated.Data

	for _, anyMessage := range data1 {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestAuthenticationSelect_IpBlocks(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthenticationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)
	policy.Action = "ALLOW"
	policy.Selector = []*dubbo_apache_org_v1alpha1.AuthenticationPolicySelector{
		{
			IpBlocks: []string{"127.0.0.0/16"},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthenticationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: pb,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127.0.0.2"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authentication.Spec.Action)
		}
	}

	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data1 := generated.Data

	for _, anyMessage := range data1 {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 0, authentication)
		}
	}
}

func TestAuthenticationSelect_NotIpBlocks_ErrFmt(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthenticationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)
	policy.Action = "ALLOW"
	policy.Selector = []*dubbo_apache_org_v1alpha1.AuthenticationPolicySelector{
		{
			NotIpBlocks: []string{"123"},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthenticationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: pb,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127.0.0.2"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authentication.Spec.Action)
		}
	}

	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127.0.0.3"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data1 := generated.Data

	for _, anyMessage := range data1 {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authentication.Spec.Action)
		}
	}
}

func TestAuthenticationSelect_Principals(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthenticationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)
	policy.Action = "ALLOW"
	policy.Selector = []*dubbo_apache_org_v1alpha1.AuthenticationPolicySelector{
		{
			Principals: []string{"cluster.local/ns/default/sa/dubbo-demo"},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthenticationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: pb,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/default/sa/dubbo-demo-new",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authentication)
		}
	}

	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/default/sa/dubbo-demo",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data1 := generated.Data

	for _, anyMessage := range data1 {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authentication.Spec.Action)
		}
	}

	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "spiffe://cluster.local/ns/default/sa/dubbo-demo",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	for _, anyMessage := range data1 {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authentication.Spec.Action)
		}
	}
}

func TestAuthenticationSelect_NotPrincipals(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthenticationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)
	policy.Action = "ALLOW"
	policy.Selector = []*dubbo_apache_org_v1alpha1.AuthenticationPolicySelector{
		{
			NotPrincipals: []string{"cluster.local/ns/default/sa/dubbo-demo"},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthenticationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: pb,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/default/sa/dubbo-demo-new",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authentication.Spec.Action)
		}
	}

	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/default/sa/dubbo-demo",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data1 := generated.Data

	for _, anyMessage := range data1 {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authentication)
		}
	}

	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "spiffe://cluster.local/ns/default/sa/dubbo-demo",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	for _, anyMessage := range data1 {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authentication)
		}
	}
}

func TestAuthenticationSelect_Extends(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthenticationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)
	policy.Action = "ALLOW"
	policy.Selector = []*dubbo_apache_org_v1alpha1.AuthenticationPolicySelector{
		{
			Extends: []*dubbo_apache_org_v1alpha1.AuthenticationPolicyExtend{
				{
					Key:   "kubernetesEnv.podName",
					Value: "dubbo-demo",
				},
			},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthenticationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: pb,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "dubbo-demo",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authentication.Spec.Action)
		}
	}

	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/default/sa/dubbo-demo",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data1 := generated.Data

	for _, anyMessage := range data1 {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authentication)
		}
	}

	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "spiffe://cluster.local/ns/default/sa/dubbo-demo",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	for _, anyMessage := range data1 {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authentication)
		}
	}
}

func TestAuthenticationSelect_NotExtends(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthenticationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)
	policy.Action = "ALLOW"
	policy.Selector = []*dubbo_apache_org_v1alpha1.AuthenticationPolicySelector{
		{
			NotExtends: []*dubbo_apache_org_v1alpha1.AuthenticationPolicyExtend{
				{
					Key:   "kubernetesEnv.podName",
					Value: "dubbo-demo",
				},
			},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthenticationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: pb,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "dubbo-demo",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 0, authentication)
		}
	}

	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "dubbo-demo-new",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthenticationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data1 := generated.Data

	for _, anyMessage := range data1 {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthenticationTypeUrl {
			authentication := &dubbo_apache_org_v1alpha1.AuthenticationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authentication)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authentication.Spec.Action)
		}
	}
}

func TestAuthorization_Empty(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthorizationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
	policy.Action = "ALLOW"
	policy.Rules = []*dubbo_apache_org_v1alpha1.AuthorizationPolicyRule{}

	origin := &OriginImpl{
		Gvk: gvk.AuthorizationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: pb,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	generated, err := origin.Exact(gen, nil)
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}
}

func TestAuthorization_Namespace(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthorizationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
	policy.Action = "ALLOW"
	policy.Rules = []*dubbo_apache_org_v1alpha1.AuthorizationPolicyRule{
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{
				Namespaces: []string{"test"},
			},
		},
		{},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthorizationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: policy,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	// success
	generated, err := origin.Exact(gen, nil)
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "test-new",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}
}

func TestAuthorization_NotNamespace(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthorizationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
	policy.Action = "ALLOW"
	policy.Rules = []*dubbo_apache_org_v1alpha1.AuthorizationPolicyRule{
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{
				NotNamespaces: []string{"test"},
			},
		},
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthorizationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: policy,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	// success
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "test-new",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "test",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}

	// success
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}
}

func TestAuthorization_IPBlocks(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthorizationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
	policy.Action = "ALLOW"
	policy.Rules = []*dubbo_apache_org_v1alpha1.AuthorizationPolicyRule{
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{
				IpBlocks: []string{"127.0.0.1/24"},
			},
		},
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthorizationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: policy,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	// success
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127.0.0.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127.0.1.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}
}

func TestAuthorization_ErrFmt(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthorizationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
	policy.Action = "ALLOW"
	policy.Rules = []*dubbo_apache_org_v1alpha1.AuthorizationPolicyRule{
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{
				IpBlocks: []string{"127"},
			},
		},
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthorizationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: policy,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	// failed
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127.0.0.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127.0.1.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}
}

func TestAuthorization_NotIPBlocks(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthorizationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
	policy.Action = "ALLOW"
	policy.Rules = []*dubbo_apache_org_v1alpha1.AuthorizationPolicyRule{
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{
				NotIpBlocks: []string{"127.0.0.1/24"},
			},
		},
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthorizationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: policy,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	// success
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127.0.1.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}

	// success
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127.0.0.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}
}

func TestAuthorization_NotIPBlocks_ErrFmt(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthorizationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
	policy.Action = "ALLOW"
	policy.Rules = []*dubbo_apache_org_v1alpha1.AuthorizationPolicyRule{
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{
				NotIpBlocks: []string{"127"},
			},
		},
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthorizationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: policy,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	// success
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127.0.1.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}

	// success
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		Ips: []string{"127.0.1.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}

	// success
	generated, err = origin.Exact(gen, &endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}
}

func TestAuthorization_Principals(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthorizationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
	policy.Action = "ALLOW"
	policy.Rules = []*dubbo_apache_org_v1alpha1.AuthorizationPolicyRule{
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{
				Principals: []string{"cluster.local/ns/default/sa/default"},
			},
		},
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthorizationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: policy,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	// success
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/default/sa/default",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}

	// success
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "spiffe://cluster.local/ns/default/sa/default",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/test/sa/default",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}
}

func TestAuthorization_NotPrincipals(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthorizationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
	policy.Action = "ALLOW"
	policy.Rules = []*dubbo_apache_org_v1alpha1.AuthorizationPolicyRule{
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{
				NotPrincipals: []string{"cluster.local/ns/default/sa/default"},
			},
		},
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthorizationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: policy,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	// success
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/test/sa/default",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}

	// success
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "spiffe://cluster.local/ns/test/sa/default",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/default/sa/default",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		SpiffeID: "spiffe://cluster.local/ns/default/sa/default",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}
}

func TestAuthorization_Extends(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthorizationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
	policy.Action = "ALLOW"
	policy.Rules = []*dubbo_apache_org_v1alpha1.AuthorizationPolicyRule{
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{
				Extends: []*dubbo_apache_org_v1alpha1.AuthorizationPolicyExtend{
					{
						Key:   "kubernetesEnv.podName",
						Value: "test",
					},
				},
			},
		},
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthorizationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: policy,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	// success
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "test",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "test-new",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}
}

func TestAuthorization_NotExtends(t *testing.T) {
	t.Parallel()

	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1AuthorizationPolicy).Build()
	r := collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource()
	configMeta := model.Meta{
		Name:             "name",
		Namespace:        "ns",
		GroupVersionKind: r.GroupVersionKind(),
	}
	pb, err := r.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	policy := pb.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
	policy.Action = "ALLOW"
	policy.Rules = []*dubbo_apache_org_v1alpha1.AuthorizationPolicyRule{
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{
				NotExtends: []*dubbo_apache_org_v1alpha1.AuthorizationPolicyExtend{
					{
						Key:   "kubernetesEnv.podName",
						Value: "test",
					},
				},
			},
		},
		{
			To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{},
		},
	}

	origin := &OriginImpl{
		Gvk: gvk.AuthorizationPolicy,
		Rev: 1,
		Data: []model.Config{
			{
				Meta: configMeta,
				Spec: policy,
			},
		},
	}
	gen := map[string]DdsResourceGenerator{}
	gen[gvk.AuthenticationPolicy] = &AuthenticationGenerator{}
	gen[gvk.AuthorizationPolicy] = &AuthorizationGenerator{}
	gen[gvk.ServiceNameMapping] = &ServiceMappingGenerator{}
	gen[gvk.ConditionRoute] = &ConditionRoutesGenerator{}
	gen[gvk.TagRoute] = &TagRoutesGenerator{}
	gen[gvk.DynamicConfig] = &DynamicConfigsGenerator{}
	// success
	generated, err := origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "test-new",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data := generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}

	// failed
	generated, err = origin.Exact(gen, &endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "test",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, nil, authorization)
		}
	}

	// success
	generated, err = origin.Exact(gen, &endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type, gvk.AuthorizationPolicy)
	assert.Equal(t, generated.Revision, int64(1))

	data = generated.Data

	for _, anyMessage := range data {
		valBytes := anyMessage.Value
		if anyMessage.TypeUrl == model.AuthorizationTypeUrl {
			authorization := &dubbo_apache_org_v1alpha1.AuthorizationPolicyToClient{}
			err := proto.Unmarshal(valBytes, authorization)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "ALLOW", authorization.Spec.Action)
		}
	}
}
