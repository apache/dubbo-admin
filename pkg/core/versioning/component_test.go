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

package versioning

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appcfg "github.com/apache/dubbo-admin/pkg/config/app"
	storecfg "github.com/apache/dubbo-admin/pkg/config/store"
	versioningcfg "github.com/apache/dubbo-admin/pkg/config/versioning"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/governor"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	corestore "github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
	_ "github.com/apache/dubbo-admin/pkg/lock/local"
)

func TestComponentEnabledFailsClosedWithoutLock(t *testing.T) {
	builder := newVersioningComponentBuilder(t, true)
	require.NoError(t, builder.ActivateComponent(newFakeVersioningRMComponent(t)))
	require.NoError(t, builder.ActivateComponent(&fakeVersioningEventBus{}))

	err := (&component{}).Init(builder)
	require.ErrorContains(t, err, "requires a lock component")
}

func TestComponentDisabledDoesNotRequireLock(t *testing.T) {
	builder := newVersioningComponentBuilder(t, false)

	c := &component{}
	require.NoError(t, c.Init(builder))
	require.NotNil(t, c.Service())
	_, err := c.Service().List(meshresource.ConditionRouteKind, "", "demo")
	require.ErrorIs(t, err, ErrFeatureDisabled)
}

func TestComponentRequiredDependenciesDoNotForceLockWhenDisabled(t *testing.T) {
	c := &component{}
	require.NotContains(t, c.RequiredDependencies(), lock.DistributedLockComponent)
}

func TestComponentMemoryStoreUsesLocalLock(t *testing.T) {
	builder := newVersioningComponentBuilder(t, true)
	require.NoError(t, builder.ActivateComponent(newFakeVersioningRMComponent(t)))
	bus := &fakeVersioningEventBus{}
	require.NoError(t, builder.ActivateComponent(bus))

	lockComp := lock.NewComponent()
	require.NoError(t, lockComp.Init(builder))
	require.NotNil(t, lockComp.GetLock())
	require.NoError(t, builder.ActivateComponent(lockComp))

	c := &component{}
	require.NoError(t, c.Init(builder))
	require.NotNil(t, c.Service())
	require.NotNil(t, c.lock)
	require.Len(t, bus.subscribers, len(governor.RuleResourceKinds.Values()))
}

func newVersioningComponentBuilder(t *testing.T, enabled bool) *runtime.Builder {
	t.Helper()
	cfg := appcfg.DefaultAdminConfig()
	cfg.Store = &storecfg.Config{Type: storecfg.Memory}
	cfg.RuleVersioning = &versioningcfg.Config{Enabled: enabled, MaxVersionsPerRule: 5}
	builder, err := runtime.BuilderFor(context.Background(), cfg)
	require.NoError(t, err)
	return builder
}

type fakeVersioningRMComponent struct {
	rm manager.ResourceManager
}

func newFakeVersioningRMComponent(t *testing.T) *fakeVersioningRMComponent {
	t.Helper()
	versionStore, intentStore, metaStore := newVersioningStores(t)
	stores := map[coremodel.ResourceKind]corestore.ResourceStore{
		meshresource.RuleVersionKind: versionStore,
		meshresource.RuleIntentKind:  intentStore,
		meshresource.RuleMetaKind:    metaStore,
	}
	return &fakeVersioningRMComponent{rm: &fakeVersioningRM{stores: stores}}
}

func (c *fakeVersioningRMComponent) Type() runtime.ComponentType { return runtime.ResourceManager }
func (c *fakeVersioningRMComponent) Order() int                  { return 0 }
func (c *fakeVersioningRMComponent) RequiredDependencies() []runtime.ComponentType {
	return nil
}
func (c *fakeVersioningRMComponent) Init(runtime.BuilderContext) error { return nil }
func (c *fakeVersioningRMComponent) Start(runtime.Runtime, <-chan struct{}) error {
	return nil
}
func (c *fakeVersioningRMComponent) ResourceManager() manager.ResourceManager { return c.rm }

type fakeVersioningRM struct {
	stores map[coremodel.ResourceKind]corestore.ResourceStore
}

func (rm *fakeVersioningRM) GetStore(kind coremodel.ResourceKind) (corestore.ResourceStore, error) {
	return rm.stores[kind], nil
}
func (rm *fakeVersioningRM) GetByKey(coremodel.ResourceKind, string) (coremodel.Resource, bool, error) {
	return nil, false, nil
}
func (rm *fakeVersioningRM) GetByKeys(coremodel.ResourceKind, []string) ([]coremodel.Resource, error) {
	return nil, nil
}
func (rm *fakeVersioningRM) ListByIndexes(coremodel.ResourceKind, []index.IndexCondition) ([]coremodel.Resource, error) {
	return nil, nil
}
func (rm *fakeVersioningRM) PageListByIndexes(coremodel.ResourceKind, []index.IndexCondition, coremodel.PageReq) (*coremodel.PageData[coremodel.Resource], error) {
	return nil, nil
}
func (rm *fakeVersioningRM) Add(context.Context, coremodel.Resource) error    { return nil }
func (rm *fakeVersioningRM) Update(context.Context, coremodel.Resource) error { return nil }
func (rm *fakeVersioningRM) Upsert(context.Context, coremodel.Resource) error { return nil }
func (rm *fakeVersioningRM) DeleteByKey(context.Context, coremodel.ResourceKind, string, string) error {
	return nil
}

type fakeVersioningEventBus struct {
	subscribers []events.Subscriber
}

func (b *fakeVersioningEventBus) Type() runtime.ComponentType { return runtime.EventBus }
func (b *fakeVersioningEventBus) Order() int                  { return 0 }
func (b *fakeVersioningEventBus) RequiredDependencies() []runtime.ComponentType {
	return nil
}
func (b *fakeVersioningEventBus) Init(runtime.BuilderContext) error { return nil }
func (b *fakeVersioningEventBus) Start(runtime.Runtime, <-chan struct{}) error {
	return nil
}
func (b *fakeVersioningEventBus) Subscribe(sub events.Subscriber) error {
	b.subscribers = append(b.subscribers, sub)
	return nil
}
func (b *fakeVersioningEventBus) Unsubscribe(events.Subscriber) error { return nil }
func (b *fakeVersioningEventBus) Send(events.Event)                   {}

var _ manager.ResourceManagerComponent = (*fakeVersioningRMComponent)(nil)
var _ manager.ResourceManager = (*fakeVersioningRM)(nil)
var _ events.EventBus = (*fakeVersioningEventBus)(nil)
var _ runtime.Component = (*fakeVersioningEventBus)(nil)
