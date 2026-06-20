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
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	storecfg "github.com/apache/dubbo-admin/pkg/config/store"
	"github.com/apache/dubbo-admin/pkg/core/events"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	locallock "github.com/apache/dubbo-admin/pkg/lock/local"
	"github.com/apache/dubbo-admin/pkg/store/dbcommon"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

func TestResourceStoreAdapter_StaleObservedAndStatusUpdatesConflictOnRevision(t *testing.T) {
	versionStore, intentStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore)

	intent, err := adapter.CreateIntent(context.Background(), testInsertRequest("demo-rule", "hash-a"))
	require.NoError(t, err)
	stale, _, err := adapter.getIntentResourceByID(intent.ID)
	require.NoError(t, err)

	require.NoError(t, adapter.MarkIntentApplied(context.Background(), intent.ID))
	err = updateIntentResourceObserved(intentStore, stale, OperationUpdate, "hash-b", `{"key":"B"}`)
	require.ErrorIs(t, err, ErrVersionIntentConflict)

	fresh, _, err := adapter.getIntentResourceByID(intent.ID)
	require.NoError(t, err)
	require.NoError(t, updateIntentResourceObserved(intentStore, fresh, OperationUpdate, "hash-b", `{"key":"B"}`))
	err = updateIntentResourceStatus(intentStore, fresh, IntentStatusCommitting, "")
	require.ErrorIs(t, err, ErrVersionIntentConflict)

	open, err := adapter.GetIntent(intent.ID)
	require.NoError(t, err)
	assert.Equal(t, IntentStatusApplied, open.Status)
	assert.True(t, open.ReconcileRequired)
	assert.Equal(t, "hash-b", open.ObservedContentHash)
}

func TestResourceStoreAdapter_GormConditionalUpdateConcurrentCommitAndObservedOnlyOneWins(t *testing.T) {
	writerA, writerB, intentStoreA, intentStoreB := newGormVersioningAdapters(t)
	key := coremodel.BuildResourceKey("", "demo-rule")

	intent, err := writerA.CreateIntent(context.Background(), testInsertRequest("demo-rule", "hash-a"))
	require.NoError(t, err)
	require.NoError(t, writerA.MarkIntentApplied(context.Background(), intent.ID))

	staleForCommit, _, err := writerA.getIntentResourceByID(intent.ID)
	require.NoError(t, err)
	staleForObserved, _, err := writerB.getIntentResourceByID(intent.ID)
	require.NoError(t, err)
	require.Equal(t, staleForCommit.Spec.Revision, staleForObserved.Spec.Revision)

	ready := make(chan struct{}, 2)
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		ready <- struct{}{}
		<-start
		results <- updateIntentResourceStatus(intentStoreA, staleForCommit, IntentStatusCommitting, "")
	}()
	go func() {
		ready <- struct{}{}
		<-start
		results <- updateIntentResourceObserved(intentStoreB, staleForObserved, OperationUpdate, "hash-b", `{"key":"B"}`)
	}()
	<-ready
	<-ready
	close(start)

	winners := 0
	conflicts := 0
	for i := 0; i < 2; i++ {
		err := <-results
		switch {
		case err == nil:
			winners++
		case errors.Is(err, ErrVersionIntentConflict):
			conflicts++
		default:
			require.NoError(t, err)
		}
	}
	require.Equal(t, 1, winners)
	require.Equal(t, 1, conflicts)

	finalIntent, err := writerA.GetIntent(intent.ID)
	require.NoError(t, err)
	switch finalIntent.Status {
	case IntentStatusCommitting:
		assert.False(t, finalIntent.ReconcileRequired)
		require.NoError(t, writerB.MarkIntentObserved(context.Background(), intent.ID, OperationUpdate, "hash-b", `{"key":"B"}`))
		_, err = writerB.CommitIntent(context.Background(), intent.ID, 10)
		require.NoError(t, err)
		versions, err := writerB.ListVersions(meshresource.ConditionRouteKind, key)
		require.NoError(t, err)
		require.Len(t, versions, 2)
		assert.Equal(t, "hash-b", versions[0].ContentHash)
		assert.Equal(t, "hash-a", versions[1].ContentHash)
	case IntentStatusApplied:
		assert.True(t, finalIntent.ReconcileRequired)
		_, err = writerB.CommitIntent(context.Background(), intent.ID, 10)
		var pending *IntentPendingError
		require.ErrorAs(t, err, &pending)
	default:
		t.Fatalf("unexpected final intent status after concurrent CAS: %s", finalIntent.Status)
	}
}

func TestResourceStoreAdapter_CommitIntentDirtyMarkerCASRejectsBeforeAppend(t *testing.T) {
	versionStore, baseIntentStore := newVersioningStores(t)
	commitAtCAS := make(chan struct{})
	releaseCommit := make(chan struct{})
	var once sync.Once
	blockingIntentStore := &barrierCASStore{ResourceStore: baseIntentStore}
	writerA := NewResourceStoreAdapter(versionStore, blockingIntentStore)
	writerB := NewResourceStoreAdapter(versionStore, baseIntentStore)

	intent, err := writerA.CreateIntent(context.Background(), testInsertRequest("demo-rule", "hash-a"))
	require.NoError(t, err)
	require.NoError(t, writerA.MarkIntentApplied(context.Background(), intent.ID))

	blockingIntentStore.beforeCAS = func(_, updated coremodel.Resource) {
		intentRes, ok := updated.(*meshresource.RuleIntentResource)
		if !ok || intentRes.Spec == nil || IntentStatus(intentRes.Spec.Status) != IntentStatusCommitting {
			return
		}
		once.Do(func() {
			close(commitAtCAS)
			<-releaseCommit
		})
	}

	errCh := make(chan error, 1)
	go func() {
		_, commitErr := writerA.CommitIntent(context.Background(), intent.ID, 10)
		errCh <- commitErr
	}()
	<-commitAtCAS

	require.NoError(t, writerB.MarkIntentObserved(context.Background(), intent.ID, OperationUpdate, "hash-b", `{"key":"B"}`))
	close(releaseCommit)

	err = <-errCh
	var pending *IntentPendingError
	require.ErrorAs(t, err, &pending)

	versions, err := writerA.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Empty(t, versions)

	open, err := writerA.GetIntent(intent.ID)
	require.NoError(t, err)
	assert.Equal(t, IntentStatusApplied, open.Status)
	assert.True(t, open.ReconcileRequired)
	assert.Equal(t, "hash-b", open.ObservedContentHash)
}

func TestSubscriber_StaleOpenIntentAfterCleanupFallsBackToUpstreamVersion(t *testing.T) {
	versionStore, baseIntentStore := newVersioningStores(t)
	intentStore := &failOnceStore{ResourceStore: baseIntentStore, err: errors.New("cleanup failed")}
	adapter := NewResourceStoreAdapter(versionStore, intentStore)
	sub := NewSubscriber(meshresource.ConditionRouteKind, adapter, 10, locallock.NewLocalLock(), context.Background())

	intended := testConditionRule("demo-rule", "A")
	req, err := buildMutationInsertRequest(intended, OperationUpdate, SourceAdmin, "admin", "", nil, time.Unix(100, 0))
	require.NoError(t, err)
	intent, err := adapter.CreateIntent(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, adapter.MarkIntentApplied(context.Background(), intent.ID))
	staleOpen, err := adapter.GetIntent(intent.ID)
	require.NoError(t, err)

	intentStore.failNextDelete = true
	_, err = adapter.CommitIntent(context.Background(), intent.ID, 10)
	require.ErrorContains(t, err, "cleanup failed")

	external := testConditionRule("demo-rule", "B")
	event, err := normalizeRuleEvent(events.NewResourceChangedEvent(cache.Updated, intended, external))
	require.NoError(t, err)
	require.NoError(t, sub.handleOpenIntentEvent(staleOpen, *event))

	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, external.ResourceKey())
	require.NoError(t, err)
	require.NotEmpty(t, versions)
	assert.Equal(t, HashSpecForTest(t, external), versions[0].ContentHash)
	assert.Equal(t, SourceUpstream, versions[0].Source)
}

func TestSubscriber_CommittingIntentFinishesAdminBeforeUpstreamSuccessor(t *testing.T) {
	baseVersionStore, baseIntentStore := newVersioningStores(t)
	intentStore := &failOnceStore{ResourceStore: baseIntentStore, err: errors.New("intent update failed")}
	adapter := NewResourceStoreAdapter(baseVersionStore, intentStore)
	sub := NewSubscriber(meshresource.ConditionRouteKind, adapter, 10, locallock.NewLocalLock(), context.Background())

	intent, err := adapter.CreateIntent(context.Background(), testInsertRequest("demo-rule", "hash-a"))
	require.NoError(t, err)
	require.NoError(t, adapter.MarkIntentApplied(context.Background(), intent.ID))
	intentStore.failUpdateAfter = 2
	_, err = adapter.CommitIntent(context.Background(), intent.ID, 10)
	require.ErrorContains(t, err, "intent update failed")

	committing, err := adapter.GetIntent(intent.ID)
	require.NoError(t, err)
	require.Equal(t, IntentStatusCommitting, committing.Status)

	upstream := testConditionRule("demo-rule", "B")
	event, err := normalizeRuleEvent(events.NewResourceChangedEvent(cache.Updated, upstream, upstream))
	require.NoError(t, err)
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Updated, upstream, upstream)))

	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, upstream.ResourceKey())
	require.NoError(t, err)
	require.Len(t, versions, 2)
	assert.Equal(t, event.ContentHash, versions[0].ContentHash)
	assert.Equal(t, SourceUpstream, versions[0].Source)
	assert.Equal(t, int64(2), versions[0].VersionNo)
	assert.Equal(t, "hash-a", versions[1].ContentHash)
	assert.Equal(t, SourceAdmin, versions[1].Source)
	assert.Equal(t, intent.ID, versions[1].ID)
	assert.Equal(t, int64(1), versions[1].VersionNo)
}

func TestService_OutcomeUnknownActualMatchCommitsFixedIntentID(t *testing.T) {
	versionStore, intentStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore)
	svc := NewService(10, adapter)
	res := testConditionRule("demo-rule", "v1")
	req, err := buildMutationInsertRequest(res, OperationUpdate, SourceAdmin, "admin", "", nil, time.Unix(100, 0))
	require.NoError(t, err)
	intent, err := adapter.CreateIntent(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, adapter.MarkIntentOutcomeUnknown(context.Background(), intent.ID, "timeout"))

	committed, err := finalizeMutationForTest(svc, intent, res, false)
	require.NoError(t, err)
	require.NotNil(t, committed)
	assert.Equal(t, intent.ID, committed.ID)
	assert.Equal(t, intent.ID, committed.IntentID)

	_, err = adapter.GetIntent(intent.ID)
	require.ErrorIs(t, err, ErrVersionIntentNotFound)
}

func TestResourceStoreAdapter_CheckExpectedVersion(t *testing.T) {
	versionStore, intentStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore)
	key := coremodel.BuildResourceKey("", "demo-rule")

	require.NoError(t, adapter.CheckExpectedVersion(meshresource.ConditionRouteKind, key, nil))

	expectedDeleted := int64(0)
	require.NoError(t, adapter.CheckExpectedVersion(meshresource.ConditionRouteKind, key, &expectedDeleted))

	version, err := adapter.InsertVersion(context.Background(), testInsertRequest("demo-rule", "hash-a"), 10)
	require.NoError(t, err)
	err = adapter.CheckExpectedVersion(meshresource.ConditionRouteKind, key, &expectedDeleted)
	var conflict *ConflictError
	require.ErrorAs(t, err, &conflict)
	require.NotNil(t, conflict.CurrentVersionID)

	mismatch := version.ID + 1
	err = adapter.CheckExpectedVersion(meshresource.ConditionRouteKind, key, &mismatch)
	require.ErrorAs(t, err, &conflict)
	require.NotNil(t, conflict.CurrentVersionID)
	assert.Equal(t, version.ID, *conflict.CurrentVersionID)

	require.NoError(t, adapter.CheckExpectedVersion(meshresource.ConditionRouteKind, key, &version.ID))
}

func TestRuleVersionLockSerializesSharedAdapters(t *testing.T) {
	versionStore, intentStore := newVersioningStores(t)
	writerA := NewResourceStoreAdapter(versionStore, intentStore)
	writerB := NewResourceStoreAdapter(versionStore, intentStore)
	lockMgr := locallock.NewLocalLock()
	key := coremodel.BuildResourceKey("", "demo-rule")

	const writes = 8
	var wg sync.WaitGroup
	errCh := make(chan error, writes)
	for i := 0; i < writes; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			adapter := writerA
			if i%2 == 1 {
				adapter = writerB
			}
			err := withRuleVersionLock(context.Background(), lockMgr, meshresource.ConditionRouteKind, key, func(leaseCtx context.Context) error {
				_, err := adapter.InsertVersion(leaseCtx, testInsertRequest("demo-rule", fmt.Sprintf("hash-%02d", i)), 100)
				return err
			})
			errCh <- err
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	versions, err := writerA.ListVersions(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Len(t, versions, writes)
	seen := map[int64]bool{}
	for i, version := range versions {
		require.False(t, seen[version.VersionNo], "duplicate VersionNo %d", version.VersionNo)
		seen[version.VersionNo] = true
		assert.Equal(t, int64(writes-i), version.VersionNo)
	}
}

func TestComponent_CleanupTerminalIntentSweepAfterRestart(t *testing.T) {
	versionStore, baseIntentStore := newVersioningStores(t)
	intentStore := &failOnceStore{ResourceStore: baseIntentStore, err: errors.New("cleanup failed")}
	adapter := NewResourceStoreAdapter(versionStore, intentStore)

	intent, err := adapter.CreateIntent(context.Background(), testInsertRequest("demo-rule", "hash-a"))
	require.NoError(t, err)
	require.NoError(t, adapter.MarkIntentApplied(context.Background(), intent.ID))

	intentStore.failNextDelete = true
	_, err = adapter.CommitIntent(context.Background(), intent.ID, 10)
	require.ErrorContains(t, err, "cleanup failed")
	terminal, err := adapter.GetIntent(intent.ID)
	require.NoError(t, err)
	require.Equal(t, IntentStatusCommitted, terminal.Status)

	c := &component{store: adapter, lock: locallock.NewLocalLock()}
	require.NoError(t, c.cleanupTerminalIntents(context.Background()))

	_, err = adapter.GetIntent(intent.ID)
	require.ErrorIs(t, err, ErrVersionIntentNotFound)
	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, intent.ID, versions[0].ID)
}

func newVersioningStores(t *testing.T) (store.ResourceStore, store.ResourceStore) {
	t.Helper()
	versionStore := memoryst.NewMemoryResourceStore(meshresource.RuleVersionKind)
	intentStore := memoryst.NewMemoryResourceStore(meshresource.RuleIntentKind)
	for _, s := range []store.ManagedResourceStore{versionStore, intentStore} {
		require.NoError(t, s.Init(nil))
	}
	return versionStore, intentStore
}

func newGormVersioningAdapters(t *testing.T) (*ResourceStoreAdapter, *ResourceStoreAdapter, store.ResourceStore, store.ResourceStore) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "versioning.db")
	dialector := sqlite.Open("file:" + dbPath + "?cache=shared&_journal_mode=WAL&_busy_timeout=5000")
	pool, err := dbcommon.NewConnectionPool(dialector, storecfg.MySQL, t.Name(), dbcommon.DefaultConnectionPoolConfig())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, pool.Close())
	})

	versionStoreA := dbcommon.NewGormStore(meshresource.RuleVersionKind, t.Name()+"-version-a", pool)
	intentStoreA := dbcommon.NewGormStore(meshresource.RuleIntentKind, t.Name()+"-intent-a", pool)
	versionStoreB := dbcommon.NewGormStore(meshresource.RuleVersionKind, t.Name()+"-version-b", pool)
	intentStoreB := dbcommon.NewGormStore(meshresource.RuleIntentKind, t.Name()+"-intent-b", pool)
	for _, s := range []store.ManagedResourceStore{versionStoreA, intentStoreA, versionStoreB, intentStoreB} {
		require.NoError(t, s.Init(nil))
	}
	return NewResourceStoreAdapter(versionStoreA, intentStoreA),
		NewResourceStoreAdapter(versionStoreB, intentStoreB),
		intentStoreA,
		intentStoreB
}

func testInsertRequest(ruleName, hash string) InsertRequest {
	return InsertRequest{
		RuleKind:    meshresource.ConditionRouteKind,
		Mesh:        "",
		ResourceKey: coremodel.BuildResourceKey("", ruleName),
		RuleName:    ruleName,
		SpecJSON:    `{"key":"` + ruleName + `","hash":"` + hash + `"}`,
		ContentHash: hash,
		Source:      SourceAdmin,
		Operation:   OperationUpdate,
		Author:      "admin",
		CreatedAt:   time.Unix(100, 0),
	}
}

func HashSpecForTest(t *testing.T, res coremodel.Resource) string {
	t.Helper()
	hash, _, err := NormalizeResource(res)
	require.NoError(t, err)
	return hash
}

func testConditionRule(ruleName, payload string) *meshresource.ConditionRouteResource {
	res := meshresource.NewConditionRouteResourceWithAttributes(ruleName, "")
	res.Spec = &meshproto.ConditionRoute{Enabled: true, Key: ruleName, Conditions: []string{payload}}
	return res
}

func finalizeMutationForTest(svc *Service, intent *Intent, current coremodel.Resource, deleted bool) (*Version, error) {
	var version *Version
	err := withRuleVersionLock(context.Background(), locallock.NewLocalLock(), intent.RuleKind, intent.ResourceKey, func(leaseCtx context.Context) error {
		var inner error
		version, inner = svc.FinalizeMutation(leaseCtx, intent, current, deleted)
		return inner
	})
	return version, err
}

type barrierCASStore struct {
	store.ResourceStore
	beforeCAS func(expected coremodel.Resource, updated coremodel.Resource)
}

func (s *barrierCASStore) UpdateIfUnchanged(expected coremodel.Resource, updated coremodel.Resource) (bool, error) {
	if s.beforeCAS != nil {
		s.beforeCAS(expected, updated)
	}
	cas, ok := s.ResourceStore.(store.ConditionalResourceStore)
	if !ok {
		return false, fmt.Errorf("wrapped store does not support conditional updates")
	}
	return cas.UpdateIfUnchanged(expected, updated)
}

type failOnceStore struct {
	store.ResourceStore
	failNextDelete  bool
	failUpdateAfter int
	err             error
}

func (s *failOnceStore) UpdateIfUnchanged(expected coremodel.Resource, updated coremodel.Resource) (bool, error) {
	if s.failUpdateAfter > 0 {
		s.failUpdateAfter--
		if s.failUpdateAfter == 0 {
			return false, s.err
		}
	}
	cas, ok := s.ResourceStore.(store.ConditionalResourceStore)
	if !ok {
		return false, fmt.Errorf("wrapped store does not support conditional updates")
	}
	return cas.UpdateIfUnchanged(expected, updated)
}

func (s *failOnceStore) Delete(obj interface{}) error {
	if s.failNextDelete {
		s.failNextDelete = false
		return s.err
	}
	return s.ResourceStore.Delete(obj)
}
