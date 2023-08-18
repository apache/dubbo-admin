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

package crdclient

import (
	"sync"
	"testing"

	dubbo_apache_org_v1alpha1 "github.com/apache/dubbo-admin/api/resource/v1alpha1"
	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core/model"
	"github.com/apache/dubbo-admin/pkg/core/schema/collections"
	"github.com/apache/dubbo-admin/pkg/dds/storage"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/util/workqueue"
)

func TestAuthentication(t *testing.T) {
	configName := "name"
	configNamespace := "namespace"
	c := collections.DubboApacheOrgV1Alpha1AuthenticationPolicy
	name := c.Resource().Kind()
	t.Run(name, func(t *testing.T) {
		r := c.Resource()
		configMeta := model.Meta{
			GroupVersionKind: r.GroupVersionKind(),
			Name:             configName,
		}
		if !r.IsClusterScoped() {
			configMeta.Namespace = configNamespace
		}

		pb, err := r.NewInstance()
		if err != nil {
			t.Fatal(err)
		}
		authenticationPolicy := pb.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)
		authenticationPolicy.Action = ""
		authenticationPolicy.Selector = []*dubbo_apache_org_v1alpha1.AuthenticationPolicySelector{
			{
				Namespaces:    []string{"test-namespace"},
				NotNamespaces: []string{"test-not-namespace"},
				IpBlocks:      []string{"test-ip-block"},
				NotIpBlocks:   []string{"test-not-ip-block"},
				Principals:    []string{"test-principal"},
				NotPrincipals: []string{"test-not-principal"},
				Extends: []*dubbo_apache_org_v1alpha1.AuthenticationPolicyExtend{
					{
						Key:   "test-key",
						Value: "test-value",
					},
				},
				NotExtends: []*dubbo_apache_org_v1alpha1.AuthenticationPolicyExtend{
					{
						Key:   "test-not-key",
						Value: "test-not-value",
					},
				},
			},
		}
		authenticationPolicy.PortLevel = []*dubbo_apache_org_v1alpha1.AuthenticationPolicyPortLevel{
			{
				Port:   1314,
				Action: "test-action",
			},
		}

		config := model.Config{
			Meta: configMeta,
			Spec: authenticationPolicy,
		}

		m := authentication(config, "rootNamespace")
		afterPolicy := m.Spec.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)
		assert.Equal(t, afterPolicy.Selector[0].Namespaces[1], config.Namespace)
	})
}

func TestAuthorization(t *testing.T) {
	configName := "name"
	configNamespace := "namespace"
	c := collections.DubboApacheOrgV1Alpha1AuthorizationPolicy
	name := c.Resource().Kind()
	t.Run(name, func(t *testing.T) {
		r := c.Resource()
		configMeta := model.Meta{
			GroupVersionKind: r.GroupVersionKind(),
			Name:             configName,
		}
		if !r.IsClusterScoped() {
			configMeta.Namespace = configNamespace
		}

		pb, err := r.NewInstance()
		if err != nil {
			t.Fatal(err)
		}
		authorizationPolicy := pb.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
		authorizationPolicy.Action = "test-action"
		authorizationPolicy.Rules = []*dubbo_apache_org_v1alpha1.AuthorizationPolicyRule{
			{
				From: &dubbo_apache_org_v1alpha1.AuthorizationPolicySource{
					Namespaces:    []string{"test-namespace"},
					NotNamespaces: []string{"test-not-namespace"},
					IpBlocks:      []string{"test-ip-block"},
					NotIpBlocks:   []string{"test-not-ip-block"},
					Principals:    []string{"test-principal"},
					NotPrincipals: []string{"test-not-principal"},
					Extends: []*dubbo_apache_org_v1alpha1.AuthorizationPolicyExtend{
						{
							Key:   "test-not-key",
							Value: "test-not-value",
						},
					},
					NotExtends: []*dubbo_apache_org_v1alpha1.AuthorizationPolicyExtend{
						{
							Key:   "test-not-key",
							Value: "test-not-value",
						},
					},
				},
				To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{
					Namespaces:    []string{"test-namespace"},
					NotNamespaces: []string{"test-not-namespace"},
					IpBlocks:      []string{"test-ip-block"},
					NotIpBlocks:   []string{"test-not-ip-block"},
					Principals:    []string{"test-principal"},
					NotPrincipals: []string{"test-not-principal"},
					Extends: []*dubbo_apache_org_v1alpha1.AuthorizationPolicyExtend{
						{
							Key:   "test-key",
							Value: "test-value",
						},
					},
					NotExtends: []*dubbo_apache_org_v1alpha1.AuthorizationPolicyExtend{
						{
							Key:   "test-key",
							Value: "test-value",
						},
					},
				},
				When: &dubbo_apache_org_v1alpha1.AuthorizationPolicyCondition{
					Key: "test-key",
					Values: []*dubbo_apache_org_v1alpha1.AuthorizationPolicyMatch{
						{
							Type:  "test-type",
							Value: "test-value",
						},
					},
					NotValues: []*dubbo_apache_org_v1alpha1.AuthorizationPolicyMatch{
						{
							Type:  "test-not-type",
							Value: "test-not-value",
						},
					},
				},
			},
		}
		authorizationPolicy.Samples = 0.5
		authorizationPolicy.Order = 0.5
		authorizationPolicy.MatchType = "test-match-type"

		config := model.Config{
			Meta: configMeta,
			Spec: authorizationPolicy,
		}

		m := authorization(config, "rootNamespace")
		afterPolicy := m.Spec.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
		assert.Equal(t, afterPolicy.Rules[0].To.Namespaces[0], configNamespace)
	})
}

func TestAuthorizationNilField(t *testing.T) {
	configName := "name"
	configNamespace := "namespace"
	c := collections.DubboApacheOrgV1Alpha1AuthorizationPolicy
	name := c.Resource().Kind()
	t.Run(name, func(t *testing.T) {
		r := c.Resource()
		configMeta := model.Meta{
			GroupVersionKind: r.GroupVersionKind(),
			Name:             configName,
		}
		if !r.IsClusterScoped() {
			configMeta.Namespace = configNamespace
		}

		pb, err := r.NewInstance()
		if err != nil {
			t.Fatal(err)
		}
		authorizationPolicy := pb.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
		authorizationPolicy.Action = "DENY"
		authorizationPolicy.Rules = []*dubbo_apache_org_v1alpha1.AuthorizationPolicyRule{
			{
				From: &dubbo_apache_org_v1alpha1.AuthorizationPolicySource{
					Namespaces: []string{"dubbo-system"},
				},
				To: &dubbo_apache_org_v1alpha1.AuthorizationPolicyTarget{
					Namespaces: []string{"ns"},
				},
			},
		}

		config := model.Config{
			Meta: configMeta,
			Spec: authorizationPolicy,
		}

		m := authorization(config, "dubbo-system")
		afterPolicy := m.Spec.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)
		assert.Equal(t, afterPolicy.Rules[0].To.Namespaces[0], configNamespace)
	})
}

func TestNotifyWithIndex(t *testing.T) {
	store := makeClient(t, collections.Rule)
	configName := "name"
	configNamespace := "namespace"
	storages := storage.NewStorage(&dubbo_cp.Config{})
	storages.Connection = append(storages.Connection, &storage.Connection{
		RawRuleQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "test-queue"),
	})
	p := &PushContext{
		rootNamespace: "",
		mutex:         &sync.Mutex{},
		revision:      map[string]int64{},
		storage:       storages,
		cache:         store,
	}
	for _, c := range collections.Rule.All() {
		name := c.Resource().Kind()
		t.Run(name, func(t *testing.T) {
			r := c.Resource()
			configMeta := model.Meta{
				GroupVersionKind: r.GroupVersionKind(),
				Name:             configName,
			}
			if !r.IsClusterScoped() {
				configMeta.Namespace = configNamespace
			}

			pb, err := r.NewInstance()
			if err != nil {
				t.Fatal(err)
			}

			if _, err := store.Create(model.Config{
				Meta: configMeta,
				Spec: pb,
			}); err != nil {
				t.Fatalf("Create(%v) => got %v", name, err)
			}

			if err := p.NotifyWithIndex(c); err != nil {
				t.Fatal(err)
			}

			connection := p.storage.Connection[0]
			item, shutdown := connection.RawRuleQueue.Get()
			if shutdown {
				t.Fatal("RawRuleQueue shut down")
			}
			gvk := c.Resource().GroupVersionKind().String()
			originafter := item.(storage.Origin)
			if originafter.Type() != gvk {
				t.Fatal("gvk is not equal")
			}
			if originafter.Revision() != p.revision[gvk] {
				t.Fatal("revision is not equal")
			}
		})
	}
}
