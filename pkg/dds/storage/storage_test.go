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

package storage_test

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/apache/dubbo-admin/pkg/config/option"

	"github.com/apache/dubbo-admin/api/dds"
	dubboapacheorgv1alpha1 "github.com/apache/dubbo-admin/api/resource/v1alpha1"
	dubbocp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/kubeclient/client"
	"github.com/apache/dubbo-admin/pkg/core/model"
	"github.com/apache/dubbo-admin/pkg/core/schema/collection"
	"github.com/apache/dubbo-admin/pkg/core/schema/collections"
	"github.com/apache/dubbo-admin/pkg/core/schema/gvk"
	"github.com/apache/dubbo-admin/pkg/dds/kube/crdclient"
	"github.com/apache/dubbo-admin/pkg/dds/storage"
	"github.com/apache/dubbo-admin/test/util/retry"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type fakeConnection struct {
	sends        []*dds.ObserveResponse
	recvChan     chan recvResult
	disconnected bool
}

type recvResult struct {
	request *dds.ObserveRequest
	err     error
}

func (f *fakeConnection) Send(targetRule *storage.VersionedRule, cr *storage.ClientStatus, response *dds.ObserveResponse) error {
	cr.LastPushedTime = time.Now().Unix()
	cr.LastPushedVersion = targetRule
	cr.LastPushNonce = response.Nonce
	cr.PushingStatus = storage.Pushing
	f.sends = append(f.sends, response)
	return nil
}

func (f *fakeConnection) Recv() (*dds.ObserveRequest, error) {
	request := <-f.recvChan

	return request.request, request.err
}

func (f *fakeConnection) Disconnect() {
	f.disconnected = true
}

func TestStorage_CloseEOF(t *testing.T) {
	t.Parallel()

	s := storage.NewStorage(&dubbocp.Config{
		Options: option.Options{
			DdsBlockMaxTime: 15000000000,
		},
	})
	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	s.Connected(&endpoint.Endpoint{
		ID: "test",
	}, fake)

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(s.Connection) != 0 {
		t.Error("expected storage to be removed")
	}
}

func TestStorage_CloseErr(t *testing.T) {
	t.Parallel()

	s := storage.NewStorage(&dubbocp.Config{
		Options: option.Options{
			DdsBlockMaxTime: 15000000000,
		},
	})
	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	s.Connected(&endpoint.Endpoint{
		ID: "test",
	}, fake)

	fake.recvChan <- recvResult{
		request: nil,
		err:     fmt.Errorf("test"),
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(s.Connection) != 0 {
		t.Error("expected storage to be removed")
	}
}

func TestStorage_UnknowType(t *testing.T) {
	t.Parallel()

	s := storage.NewStorage(&dubbocp.Config{
		Options: option.Options{
			DdsBlockMaxTime: 15000000000,
		},
	})
	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	s.Connected(&endpoint.Endpoint{
		ID: "test",
	}, fake)

	fake.recvChan <- recvResult{
		request: &dds.ObserveRequest{
			Nonce: "",
			Type:  "test",
		},
		err: nil,
	}

	fake.recvChan <- recvResult{
		request: &dds.ObserveRequest{
			Nonce: "",
			Type:  "",
		},
		err: nil,
	}

	conn := s.Connection[0]

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(conn.TypeListened) != 0 {
		t.Error("expected no type listened")
	}
}

func TestStorage_StartNonEmptyNonce(t *testing.T) {
	t.Parallel()

	s := storage.NewStorage(&dubbocp.Config{
		Options: option.Options{
			DdsBlockMaxTime: 15000000000,
		},
	})
	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	s.Connected(&endpoint.Endpoint{
		ID: "test",
	}, fake)

	fake.recvChan <- recvResult{
		request: &dds.ObserveRequest{
			Nonce: "test",
			Type:  gvk.AuthenticationPolicy,
		},
		err: nil,
	}

	conn := s.Connection[0]
	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(conn.TypeListened) != 0 {
		t.Error("expected no type listened")
	}
}

func TestStorage_Listen(t *testing.T) {
	t.Parallel()

	s := storage.NewStorage(&dubbocp.Config{
		Options: option.Options{
			DdsBlockMaxTime: 15000000000,
		},
	})
	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	s.Connected(&endpoint.Endpoint{
		ID: "test",
	}, fake)

	fake.recvChan <- recvResult{
		request: &dds.ObserveRequest{
			Nonce: "",
			Type:  gvk.AuthorizationPolicy,
		},
		err: nil,
	}

	conn := s.Connection[0]

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(conn.TypeListened) == 0 {
		t.Error("expected type listened")
	}

	if !conn.TypeListened[gvk.AuthorizationPolicy] {
		t.Error("expected type listened")
	}
}

func makeClient(t *testing.T, schemas collection.Schemas) crdclient.ConfigStoreCache {
	fake := client.NewFakeClient()
	for _, s := range schemas.All() {
		_, err := fake.Ext().ApiextensionsV1().CustomResourceDefinitions().Create(context.TODO(), &v1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s.%s", s.Resource().Plural(), s.Resource().Group()),
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return nil
		}
	}
	stop := make(chan struct{})
	config, err := crdclient.New(fake, "")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		err := config.Start(stop)
		if err != nil {
			t.Error(err)
			return
		}
	}()
	_ = fake.Start(stop)
	cache.WaitForCacheSync(stop, config.HasSynced)
	t.Cleanup(func() {
		close(stop)
	})
	return config
}

func TestStorage_PreNotify(t *testing.T) {
	t.Parallel()

	store := makeClient(t, collections.Rule)
	configName := "name"
	configNamespace := "namespace"
	timeout := retry.Timeout(time.Second * 20)
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
			// Kubernetes is eventually consistent, so we allow a short time to pass before we get
			retry.UntilSuccessOrFail(t, func() error {
				cfg := store.Get(r.GroupVersionKind(), configName, configMeta.Namespace)
				if cfg == nil || !reflect.DeepEqual(cfg.Meta, configMeta) {
					return fmt.Errorf("get(%v) => got unexpected object %v", name, cfg)
				}
				return nil
			}, timeout)
			s := storage.NewStorage(&dubbocp.Config{
				Options: option.Options{
					DdsBlockMaxTime: 15000000000,
				},
			})

			handler := crdclient.NewHandler(s, "dubbo-demo", store)
			err = handler.NotifyWithIndex(c)
			if err != nil {
				t.Fatal(err)
			}

			fake := &fakeConnection{
				recvChan: make(chan recvResult, 1),
			}

			s.Connected(&endpoint.Endpoint{
				ID: "test",
			}, fake)

			fake.recvChan <- recvResult{
				request: &dds.ObserveRequest{
					Nonce: "",
					Type:  c.Resource().GroupVersionKind().String(),
				},
				err: nil,
			}

			assert.Eventually(t, func() bool {
				return len(fake.sends) == 1
			}, 10*time.Second, time.Millisecond)

			if fake.sends[0].Type != c.Resource().GroupVersionKind().String() {
				t.Error("expected rule type")
			}

			if fake.sends[0].Nonce == "" {
				t.Error("expected non empty nonce")
			}

			if fake.sends[0].Data == nil {
				t.Error("expected data")
			}

			if fake.sends[0].Revision != 1 {
				t.Error("expected Rev 1")
			}

			fake.recvChan <- recvResult{
				request: &dds.ObserveRequest{
					Nonce: fake.sends[0].Nonce,
					Type:  c.Resource().GroupVersionKind().String(),
				},
				err: nil,
			}

			conn := s.Connection[0]

			assert.Eventually(t, func() bool {
				return conn.ClientRules[c.Resource().GroupVersionKind().String()].PushingStatus == storage.Pushed
			}, 10*time.Second, time.Millisecond)

			fake.recvChan <- recvResult{
				request: nil,
				err:     io.EOF,
			}

			assert.Eventually(t, func() bool {
				return fake.disconnected
			}, 10*time.Second, time.Millisecond)

			if len(conn.TypeListened) == 0 {
				t.Error("expected type listened")
			}

			if !conn.TypeListened[c.Resource().GroupVersionKind().String()] {
				t.Error("expected type listened")
			}
		})
	}
}

func TestStorage_AfterNotify(t *testing.T) {
	t.Parallel()

	store := makeClient(t, collections.Rule)
	configName := "name"
	configNamespace := "namespace"
	timeout := retry.Timeout(time.Second * 20)
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
			if r.GroupVersionKind().String() == gvk.ServiceNameMapping {
				mapping := pb.(*dubboapacheorgv1alpha1.ServiceNameMapping)
				mapping.InterfaceName = "test"
				mapping.ApplicationNames = []string{
					"test-app",
				}
			}
			if _, err := store.Create(model.Config{
				Meta: configMeta,
				Spec: pb,
			}); err != nil {
				t.Fatalf("Create(%v) => got %v", name, err)
			}
			// Kubernetes is eventually consistent, so we allow a short time to pass before we get
			retry.UntilSuccessOrFail(t, func() error {
				cfg := store.Get(r.GroupVersionKind(), configName, configMeta.Namespace)
				if cfg == nil || !reflect.DeepEqual(cfg.Meta, configMeta) {
					return fmt.Errorf("get(%v) => got unexpected object %v", name, cfg)
				}
				return nil
			}, timeout)
			s := storage.NewStorage(&dubbocp.Config{
				Options: option.Options{
					DdsBlockMaxTime: 15000000000,
				},
			})
			handler := crdclient.NewHandler(s, "dubbo-demo", store)

			fake := &fakeConnection{
				recvChan: make(chan recvResult, 1),
			}

			s.Connected(&endpoint.Endpoint{
				ID: "test",
			}, fake)

			fake.recvChan <- recvResult{
				request: &dds.ObserveRequest{
					Nonce: "",
					Type:  c.Resource().GroupVersionKind().String(),
				},
				err: nil,
			}

			conn := s.Connection[0]

			assert.Eventually(t, func() bool {
				return conn.TypeListened[c.Resource().GroupVersionKind().String()]
			}, 10*time.Second, time.Millisecond)

			err = handler.NotifyWithIndex(c)
			if err != nil {
				t.Fatal(err)
			}

			assert.Eventually(t, func() bool {
				return len(fake.sends) == 1
			}, 10*time.Second, time.Millisecond)

			if fake.sends[0].Type != c.Resource().GroupVersionKind().String() {
				t.Error("expected rule type")
			}

			if fake.sends[0].Nonce == "" {
				t.Error("expected non empty nonce")
			}

			if fake.sends[0].Data == nil {
				t.Error("expected data")
			}

			if fake.sends[0].Revision != 1 {
				t.Error("expected Rev 1")
			}

			fake.recvChan <- recvResult{
				request: &dds.ObserveRequest{
					Nonce: fake.sends[0].Nonce,
					Type:  c.Resource().GroupVersionKind().String(),
				},
				err: nil,
			}

			assert.Eventually(t, func() bool {
				return conn.ClientRules[c.Resource().GroupVersionKind().String()].PushingStatus == storage.Pushed
			}, 10*time.Second, time.Millisecond)

			fake.recvChan <- recvResult{
				request: nil,
				err:     io.EOF,
			}

			assert.Eventually(t, func() bool {
				return fake.disconnected
			}, 10*time.Second, time.Millisecond)

			if len(conn.TypeListened) == 0 {
				t.Error("expected type listened")
			}

			if !conn.TypeListened[c.Resource().GroupVersionKind().String()] {
				t.Error("expected type listened")
			}
		})
	}
}

func TestStore_MissNotify(t *testing.T) {
	t.Parallel()

	store := makeClient(t, collections.Rule)
	configName := "name"
	configNamespace := "namespace"
	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1TagRoute).Build()
	tag := collections.DubboApacheOrgV1Alpha1TagRoute.Resource()
	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1ConditionRoute).Build()
	condition := collections.DubboApacheOrgV1Alpha1ConditionRoute.Resource()
	tagconfigMeta := model.Meta{
		GroupVersionKind: tag.GroupVersionKind(),
		Name:             configName,
	}
	conditionConfigMeta := model.Meta{
		GroupVersionKind: condition.GroupVersionKind(),
		Name:             configName,
	}
	if !tag.IsClusterScoped() {
		tagconfigMeta.Namespace = configNamespace
	}

	tagpb, err := tag.NewInstance()
	if err != nil {
		t.Fatal(err)
	}
	conditionpb, err := condition.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Create(model.Config{
		Meta: conditionConfigMeta,
		Spec: conditionpb,
	}); err != nil {
		t.Fatalf("Create(%v) => got %v", condition.Kind(), err)
	}

	if _, err := store.Create(model.Config{
		Meta: tagconfigMeta,
		Spec: tagpb,
	}); err != nil {
		t.Fatalf("Create(%v) => got %v", tag.Kind(), err)
	}

	s := storage.NewStorage(&dubbocp.Config{
		Options: option.Options{
			DdsBlockMaxTime: 15000000000,
		},
	})
	tagHanlder := crdclient.NewHandler(s, "dubbo-demo", store)
	conditionHandler := crdclient.NewHandler(s, "dubbo-demo", store)

	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	s.Connected(&endpoint.Endpoint{
		ID: "test",
	}, fake)

	fake.recvChan <- recvResult{
		request: &dds.ObserveRequest{
			Nonce: "",
			Type:  condition.GroupVersionKind().String(),
		},
		err: nil,
	}

	conn := s.Connection[0]

	assert.Eventually(t, func() bool {
		return conn.TypeListened[condition.GroupVersionKind().String()]
	}, 10*time.Second, time.Millisecond)

	if err := conditionHandler.NotifyWithIndex(collections.DubboApacheOrgV1Alpha1ConditionRoute); err != nil {
		t.Fatal(err)
	}
	if err := tagHanlder.NotifyWithIndex(collections.DubboApacheOrgV1Alpha1TagRoute); err != nil {
		t.Fatal(err)
	}

	assert.Eventually(t, func() bool {
		return len(fake.sends) == 1
	}, 10*time.Second, time.Millisecond)

	if fake.sends[0].Type != condition.GroupVersionKind().String() {
		t.Error("expected rule type")
	}

	if fake.sends[0].Nonce == "" {
		t.Error("expected non empty nonce")
	}

	if fake.sends[0].Data == nil {
		t.Error("expected data")
	}

	if fake.sends[0].Revision != 1 {
		t.Error("expected Rev 1")
	}

	fake.recvChan <- recvResult{
		request: &dds.ObserveRequest{
			Nonce: fake.sends[0].Nonce,
			Type:  condition.GroupVersionKind().String(),
		},
		err: nil,
	}

	assert.Eventually(t, func() bool {
		return conn.ClientRules[condition.GroupVersionKind().String()].PushingStatus == storage.Pushed
	}, 10*time.Second, time.Millisecond)

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(conn.TypeListened) == 0 {
		t.Error("expected type listened")
	}

	if !conn.TypeListened[condition.GroupVersionKind().String()] {
		t.Error("expected type listened")
	}

	if len(fake.sends) != 1 {
		t.Error("expected 1 send")
	}
}

type fakeOrigin struct {
	hash int
}

func (f *fakeOrigin) Type() string {
	return gvk.TagRoute
}

func (f *fakeOrigin) Revision() int64 {
	return 1
}

func (f *fakeOrigin) Exact(gen map[string]storage.DdsResourceGenerator, endpoint *endpoint.Endpoint) (*storage.VersionedRule, error) {
	return &storage.VersionedRule{
		Type:     gvk.TagRoute,
		Revision: 1,
		Data:     []*anypb.Any{},
	}, nil
}

type errOrigin struct{}

func (e errOrigin) Type() string {
	return gvk.TagRoute
}

func (e errOrigin) Revision() int64 {
	return 1
}

func (e errOrigin) Exact(gen map[string]storage.DdsResourceGenerator, endpoint *endpoint.Endpoint) (*storage.VersionedRule, error) {
	return nil, fmt.Errorf("test")
}

func TestStorage_MulitiNotify(t *testing.T) {
	t.Parallel()

	s := storage.NewStorage(&dubbocp.Config{
		Options: option.Options{
			DdsBlockMaxTime: 15000000000,
		},
	})
	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	s.Connected(&endpoint.Endpoint{
		ID: "test",
	}, fake)

	fake.recvChan <- recvResult{
		request: &dds.ObserveRequest{
			Nonce: "",
			Type:  gvk.TagRoute,
		},
		err: nil,
	}

	conn := s.Connection[0]

	assert.Eventually(t, func() bool {
		return conn.TypeListened[gvk.TagRoute]
	}, 10*time.Second, time.Millisecond)

	// should err
	conn.RawRuleQueue.Add(&errOrigin{})
	conn.RawRuleQueue.Add(&fakeOrigin{
		hash: 1,
	})
	conn.RawRuleQueue.Add(&fakeOrigin{
		hash: 2,
	})
	conn.RawRuleQueue.Add(&fakeOrigin{
		hash: 3,
	})

	assert.Eventually(t, func() bool {
		return len(fake.sends) == 1
	}, 10*time.Second, time.Millisecond)

	if fake.sends[0].Type != gvk.TagRoute {
		t.Error("expected rule type")
	}

	if fake.sends[0].Nonce == "" {
		t.Error("expected non empty nonce")
	}

	if fake.sends[0].Data == nil {
		t.Error("expected data")
	}

	assert.Eventually(t, func() bool {
		return conn.ClientRules[gvk.TagRoute].PushQueued
	}, 10*time.Second, time.Millisecond)

	fake.recvChan <- recvResult{
		request: &dds.ObserveRequest{
			Nonce: fake.sends[0].Nonce,
			Type:  gvk.TagRoute,
		},
		err: nil,
	}
	assert.Eventually(t, func() bool {
		return conn.ClientRules[gvk.TagRoute].PushingStatus == storage.Pushed
	}, 10*time.Second, time.Millisecond)

	assert.Eventually(t, func() bool {
		return conn.RawRuleQueue.Len() == 0
	}, 10*time.Second, time.Millisecond)

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(conn.TypeListened) == 0 {
		t.Error("expected type listened")
	}

	if !conn.TypeListened[gvk.TagRoute] {
		t.Error("expected type listened")
	}

	if len(fake.sends) != 1 {
		t.Error("expected 1 send")
	}
}

func TestStorage_Exact(t *testing.T) {
	t.Parallel()

	configName := "name"
	configNamespace := "namespace"
	for _, c := range collections.Rule.All() {
		r := c.Resource()
		name := c.Resource().Kind()
		t.Run(name, func(t *testing.T) {
			configMeta := model.Meta{
				Name:             configName,
				Namespace:        configNamespace,
				GroupVersionKind: r.GroupVersionKind(),
			}

			if !r.IsClusterScoped() {
				configMeta.Namespace = configNamespace
			}

			pb, err := r.NewInstance()
			if err != nil {
				t.Fatal(err)
			}

			if r.GroupVersionKind().String() == gvk.TagRoute {
				route := pb.(*dubboapacheorgv1alpha1.TagRoute)
				route.Key = "test-key"
				route.Tags = []*dubboapacheorgv1alpha1.Tag{
					{
						Name: "zyq",
						Addresses: []string{
							"lxy",
						},
					},
				}
			}

			origin := &storage.OriginImpl{
				Gvk: r.GroupVersionKind().String(),
				Rev: 1,
				Data: []model.Config{
					{
						Meta: configMeta,
						Spec: pb,
					},
				},
			}

			gen := map[string]storage.DdsResourceGenerator{}
			gen[gvk.AuthenticationPolicy] = &storage.AuthenticationGenerator{}
			gen[gvk.AuthorizationPolicy] = &storage.AuthorizationGenerator{}
			gen[gvk.ServiceNameMapping] = &storage.ServiceMappingGenerator{}
			gen[gvk.ConditionRoute] = &storage.ConditionRoutesGenerator{}
			gen[gvk.TagRoute] = &storage.TagRoutesGenerator{}
			gen[gvk.DynamicConfig] = &storage.DynamicConfigsGenerator{}
			generated, err := origin.Exact(gen, &endpoint.Endpoint{})
			assert.Nil(t, err)

			assert.NotNil(t, generated)
			assert.Equal(t, generated.Type, r.GroupVersionKind().String())
			assert.Equal(t, generated.Revision, int64(1))
		})
	}
}

func TestStorage_ReturnMisNonce(t *testing.T) {
	t.Parallel()

	store := makeClient(t, collections.Rule)
	configName := "name"
	configNamespace := "namespace"
	collection.NewSchemasBuilder().MustAdd(collections.DubboApacheOrgV1Alpha1TagRoute).Build()
	tag := collections.DubboApacheOrgV1Alpha1TagRoute.Resource()
	tagconfigMeta := model.Meta{
		GroupVersionKind: tag.GroupVersionKind(),
		Name:             configName,
	}

	if !tag.IsClusterScoped() {
		tagconfigMeta.Namespace = configNamespace
	}

	tagpb, err := tag.NewInstance()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Create(model.Config{
		Meta: tagconfigMeta,
		Spec: tagpb,
	}); err != nil {
		t.Fatalf("Create(%v) => got %v", tag.Kind(), err)
	}

	s := storage.NewStorage(&dubbocp.Config{
		Options: option.Options{
			DdsBlockMaxTime: 15000000000,
		},
	})
	tagHanlder := crdclient.NewHandler(s, "dubbo-system", store)
	err = tagHanlder.NotifyWithIndex(collections.DubboApacheOrgV1Alpha1TagRoute)
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	s.Connected(&endpoint.Endpoint{
		ID: "TEST",
	}, fake)

	fake.recvChan <- recvResult{
		request: &dds.ObserveRequest{
			Nonce: "",
			Type:  gvk.TagRoute,
		},
		err: nil,
	}

	assert.Eventually(t, func() bool {
		return len(fake.sends) == 1
	}, 10*time.Second, time.Millisecond)

	if fake.sends[0].Type != gvk.TagRoute {
		t.Error("expected rule type")
	}
	if fake.sends[0].Nonce == "" {
		t.Error("expected non empty nonce")
	}

	if fake.sends[0].Data == nil {
		t.Error("expected data")
	}
	if fake.sends[0].Revision != 1 {
		t.Error("expected revision 1")
	}

	fake.recvChan <- recvResult{
		request: &dds.ObserveRequest{
			Nonce: "test",
			Type:  gvk.TagRoute,
		},
		err: nil,
	}

	conn := s.Connection[0]

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(conn.TypeListened) == 0 {
		t.Error("expected type listened")
	}

	if !conn.TypeListened[gvk.TagRoute] {
		t.Error("expected type listened")
	}

	if conn.ClientRules[gvk.TagRoute].PushingStatus == storage.Pushed {
		t.Error("expected not pushed")
	}
}
