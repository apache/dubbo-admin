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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appcfg "github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/config/mode"
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
	locallock "github.com/apache/dubbo-admin/pkg/lock/local"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

func TestComponentFailsClosedWithoutLock(t *testing.T) {
	builder := newVersioningComponentBuilder(t)
	require.NoError(t, builder.ActivateComponent(newFakeVersioningRMComponent(t)))
	require.NoError(t, builder.ActivateComponent(&fakeVersioningEventBus{}))

	err := (&component{}).Init(builder)
	require.ErrorContains(t, err, "requires a lock component")
}

func TestComponentRequiredDependenciesDoNotForceLockDependency(t *testing.T) {
	c := &component{}
	require.NotContains(t, c.RequiredDependencies(), lock.DistributedLockComponent)
}

func TestComponentMemoryStoreUsesLocalLock(t *testing.T) {
	builder := newVersioningComponentBuilder(t)
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

func TestComponentRepairOpenIntentsHonorsCancellationWhileWaitingForLock(t *testing.T) {
	versionStore, intentStore, _ := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore)
	intent, err := adapter.CreateIntent(context.Background(), testInsertRequest("repair-cancel-rule", "hash-a"))
	require.NoError(t, err)

	lockMgr := locallock.NewLocalLock()
	lease := holdRuleVersionLock(t, lockMgr, intent.RuleKind, intent.ResourceKey)
	defer func() { require.NoError(t, lease.Unlock(context.Background())) }()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	c := &component{
		service: NewService(5, adapter),
		store:   adapter,
		lock:    lockMgr,
	}
	err = c.repairOpenIntents(ctx, &fakeVersioningRM{})
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestComponentBootstrapExistingRulesHonorsCancellationWhileWaitingForLock(t *testing.T) {
	versionStore, intentStore, _ := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore)
	res := testConditionRule("bootstrap-cancel-rule", "v1")
	conditionStore := newRuleStoreWithResource(t, meshresource.ConditionRouteKind, res)
	rm := &fakeVersioningRM{stores: map[coremodel.ResourceKind]corestore.ResourceStore{
		meshresource.ConditionRouteKind: conditionStore,
	}}

	lockMgr := locallock.NewLocalLock()
	lease := holdRuleVersionLock(t, lockMgr, res.ResourceKind(), res.ResourceKey())
	defer func() { require.NoError(t, lease.Unlock(context.Background())) }()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	c := &component{
		service: NewService(5, adapter),
		store:   adapter,
		lock:    lockMgr,
	}
	err := c.bootstrapExistingRules(ctx, rm, 5)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	assert.Empty(t, versions)
}

func TestComponentStartHonorsStopDuringBootstrapLockWait(t *testing.T) {
	versionStore, intentStore, _ := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore)
	res := testConditionRule("bootstrap-stop-rule", "v1")
	conditionStore := newRuleStoreWithResource(t, meshresource.ConditionRouteKind, res)
	rmComp := &fakeVersioningRMComponent{rm: &fakeVersioningRM{stores: map[coremodel.ResourceKind]corestore.ResourceStore{
		meshresource.ConditionRouteKind: conditionStore,
	}}}

	lockMgr := locallock.NewLocalLock()
	lease := holdRuleVersionLock(t, lockMgr, res.ResourceKind(), res.ResourceKey())
	defer func() { require.NoError(t, lease.Unlock(context.Background())) }()

	cfg := appcfg.DefaultAdminConfig()
	cfg.RuleVersioning = &versioningcfg.Config{MaxVersionsPerRule: 5}
	rt := &fakeVersioningRuntime{
		cfg: appcfg.DefaultAdminConfig(),
		components: map[runtime.ComponentType]runtime.Component{
			runtime.ResourceManager: rmComp,
		},
		appCtx: context.Background(),
	}
	rt.cfg = cfg
	c := &component{
		service: NewService(5, adapter),
		store:   adapter,
		lock:    lockMgr,
	}
	stop := make(chan struct{})
	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Start(rt, stop)
	}()
	time.AfterFunc(20*time.Millisecond, func() {
		close(stop)
	})

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("versioning Start did not return after stop closed")
	}
}

func TestComponentBootstrapExistingRulesIsIdempotentAcrossRestarts(t *testing.T) {
	versionStore, intentStore, _ := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore)
	res := testConditionRule("bootstrap-idempotent-rule", "v1")
	conditionStore := newRuleStoreWithResource(t, meshresource.ConditionRouteKind, res)
	rm := &fakeVersioningRM{stores: map[coremodel.ResourceKind]corestore.ResourceStore{
		meshresource.ConditionRouteKind: conditionStore,
	}}

	c := &component{
		service: NewService(5, adapter),
		store:   adapter,
		lock:    locallock.NewLocalLock(),
	}
	require.NoError(t, c.bootstrapExistingRules(context.Background(), rm, 5))
	require.NoError(t, c.bootstrapExistingRules(context.Background(), rm, 5))

	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, SourceBootstrap, versions[0].Source)
	assert.Equal(t, int64(1), versions[0].VersionNo)
}

func newVersioningComponentBuilder(t *testing.T) *runtime.Builder {
	t.Helper()
	cfg := appcfg.DefaultAdminConfig()
	cfg.Store = &storecfg.Config{Type: storecfg.Memory}
	cfg.RuleVersioning = &versioningcfg.Config{MaxVersionsPerRule: 5}
	builder, err := runtime.BuilderFor(context.Background(), cfg)
	require.NoError(t, err)
	return builder
}

type fakeVersioningRMComponent struct {
	rm manager.ResourceManager
}

func newFakeVersioningRMComponent(t *testing.T) *fakeVersioningRMComponent {
	t.Helper()
	versionStore, intentStore, _ := newVersioningStores(t)
	stores := map[coremodel.ResourceKind]corestore.ResourceStore{
		meshresource.RuleVersionKind: versionStore,
		meshresource.RuleIntentKind:  intentStore,
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
func (rm *fakeVersioningRM) GetByKey(kind coremodel.ResourceKind, key string) (coremodel.Resource, bool, error) {
	rs := rm.stores[kind]
	if rs == nil {
		return nil, false, nil
	}
	item, exists, err := rs.GetByKey(key)
	if err != nil || !exists {
		return nil, exists, err
	}
	res, ok := item.(coremodel.Resource)
	if !ok {
		return nil, false, nil
	}
	return res, true, nil
}
func (rm *fakeVersioningRM) GetByKeys(kind coremodel.ResourceKind, keys []string) ([]coremodel.Resource, error) {
	rs := rm.stores[kind]
	if rs == nil {
		return nil, nil
	}
	return rs.GetByKeys(keys)
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

type fakeVersioningRuntime struct {
	cfg        appcfg.AdminConfig
	components map[runtime.ComponentType]runtime.Component
	appCtx     context.Context
}

func (rt *fakeVersioningRuntime) GetInstanceId() string { return "test-instance" }
func (rt *fakeVersioningRuntime) GetClusterId() string  { return "test-cluster" }
func (rt *fakeVersioningRuntime) GetStartTime() time.Time {
	return time.Unix(0, 0)
}
func (rt *fakeVersioningRuntime) GetMode() mode.Mode { return mode.Test }
func (rt *fakeVersioningRuntime) Config() appcfg.AdminConfig {
	return rt.cfg
}
func (rt *fakeVersioningRuntime) GetComponent(typ runtime.ComponentType) (runtime.Component, error) {
	comp := rt.components[typ]
	if comp == nil {
		return nil, assert.AnError
	}
	return comp, nil
}
func (rt *fakeVersioningRuntime) AppContext() context.Context {
	if rt.appCtx == nil {
		return context.Background()
	}
	return rt.appCtx
}
func (rt *fakeVersioningRuntime) Add(components ...runtime.Component) {
	for _, comp := range components {
		rt.components[comp.Type()] = comp
	}
}
func (rt *fakeVersioningRuntime) Start(<-chan struct{}) error { return nil }

func newRuleStoreWithResource(t *testing.T, kind coremodel.ResourceKind, res coremodel.Resource) corestore.ResourceStore {
	t.Helper()
	rs := memoryst.NewMemoryResourceStore(kind)
	require.NoError(t, rs.Init(nil))
	require.NoError(t, rs.Add(res))
	return rs
}

func holdRuleVersionLock(t *testing.T, lockMgr lock.Lock, kind coremodel.ResourceKind, resourceKey string) lock.Lease {
	t.Helper()
	key := lock.BuildRuleVersioningLockKey(string(kind), extractMesh(resourceKey), extractName(resourceKey))
	lease, err := lockMgr.Acquire(context.Background(), key, time.Second)
	require.NoError(t, err)
	return lease
}
