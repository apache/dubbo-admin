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
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/events"
	corelock "github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
	locallock "github.com/apache/dubbo-admin/pkg/lock/local"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

func TestResourceStoreAdapter_GetIntentUsesIDIndex(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, &noListKeysStore{ResourceStore: intentStore, t: t}, metaStore)

	intent, err := adapter.CreateIntent(context.Background(), testInsertRequest("demo-rule", "hash-a"))
	require.NoError(t, err)

	got, err := adapter.GetIntent(intent.ID)
	require.NoError(t, err)
	assert.Equal(t, intent.ID, got.ID)
	assert.Equal(t, "hash-a", got.ContentHash)
}

func TestResourceStoreAdapter_GetIntentMissing(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)

	_, err := adapter.GetIntent(404)
	require.ErrorIs(t, err, ErrVersionIntentNotFound)
}

func TestResourceStoreAdapter_GetIntentDuplicateIDIndex(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)

	require.NoError(t, intentStore.Add(testIntentResource("demo-a", 42, IntentStatusPending, "hash-a")))
	require.NoError(t, intentStore.Add(testIntentResource("demo-b", 42, IntentStatusPending, "hash-b")))

	_, err := adapter.GetIntent(42)
	require.ErrorIs(t, err, ErrVersionLedgerCorrupt)
}

func TestResourceStoreAdapter_IntentStatusUpdatesUseID(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)

	intent, err := adapter.CreateIntent(context.Background(), testInsertRequest("demo-rule", "hash-a"))
	require.NoError(t, err)
	require.NoError(t, adapter.MarkIntentApplied(context.Background(), intent.ID))

	applied, err := adapter.GetIntent(intent.ID)
	require.NoError(t, err)
	assert.Equal(t, IntentStatusApplied, applied.Status)

	version, err := adapter.CommitIntent(context.Background(), intent.ID, 10)
	require.NoError(t, err)
	assert.Equal(t, intent.ID, version.ID)
	assert.Equal(t, intent.ID, version.IntentID)

	_, err = adapter.GetIntent(intent.ID)
	require.ErrorIs(t, err, ErrVersionIntentNotFound)

	meta, err := adapter.ReconcileMeta(context.Background(), meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.NotNil(t, meta.CurrentVersion)
	assert.Equal(t, intent.ID, *meta.CurrentVersion)

	failedIntent, err := adapter.CreateIntent(context.Background(), testInsertRequest("demo-rule", "hash-b"))
	require.NoError(t, err)
	require.NoError(t, adapter.MarkIntentFailed(context.Background(), failedIntent.ID, "mutation failed"))

	_, err = adapter.GetIntent(failedIntent.ID)
	require.ErrorIs(t, err, ErrVersionIntentNotFound)
}

func TestResourceStoreAdapter_InsertVersionFixedIDRepairAfterMetaFailure(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	failingMetaStore := &failOnceStore{ResourceStore: metaStore, failNextAdd: true, err: errors.New("meta add failed")}
	adapter := NewResourceStoreAdapter(versionStore, intentStore, failingMetaStore)
	svc := NewService(true, 10, adapter)

	res := testConditionRule("demo-rule", "v1")
	req, err := buildMutationInsertRequest(res, OperationUpdate, SourceAdmin, "admin", "", nil, time.Unix(100, 0))
	require.NoError(t, err)
	intent, err := adapter.CreateIntent(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, adapter.MarkIntentApplied(context.Background(), intent.ID))

	_, err = adapter.CommitIntent(context.Background(), intent.ID, 10)
	require.ErrorContains(t, err, "failed to reconcile meta")

	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, intent.ID, versions[0].ID)

	var repaired *Version
	err = withRuleVersionLock(locallock.NewLocalLock(), meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"), func(leaseCtx context.Context) error {
		var inner error
		repaired, inner = svc.FinalizeMutation(leaseCtx, intent, res, false)
		return inner
	})
	require.NoError(t, err)
	require.NotNil(t, repaired)
	assert.Equal(t, intent.ID, repaired.ID)
	assert.Equal(t, intent.ID, repaired.IntentID)

	versions, err = adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, int64(1), versions[0].VersionNo)

	meta, err := adapter.ReconcileMeta(context.Background(), meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.NotNil(t, meta.CurrentVersion)
	assert.Equal(t, intent.ID, *meta.CurrentVersion)
	assert.Equal(t, int64(1), meta.LastVersionNo)
}

func TestResourceStoreAdapter_CommitIntentRetryAfterIntentStatusFailure(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	failingIntentStore := &failOnceStore{ResourceStore: intentStore, err: errors.New("intent status update failed")}
	adapter := NewResourceStoreAdapter(versionStore, failingIntentStore, metaStore)

	intent, err := adapter.CreateIntent(context.Background(), testInsertRequest("demo-rule", "hash-a"))
	require.NoError(t, err)
	require.NoError(t, adapter.MarkIntentApplied(context.Background(), intent.ID))

	failingIntentStore.failNextUpdate = true
	_, err = adapter.CommitIntent(context.Background(), intent.ID, 10)
	require.ErrorContains(t, err, "intent status update failed")

	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, intent.ID, versions[0].ID)
	assert.Equal(t, intent.ID, versions[0].IntentID)

	committed, err := adapter.CommitIntent(context.Background(), intent.ID, 10)
	require.NoError(t, err)
	assert.Equal(t, intent.ID, committed.ID)
	assert.Equal(t, int64(1), committed.VersionNo)

	versions, err = adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, int64(1), versions[0].VersionNo)

	_, err = adapter.GetIntent(intent.ID)
	require.ErrorIs(t, err, ErrVersionIntentNotFound)
}

func TestResourceStoreAdapter_FixedVersionRetryKeepsVersionNoAfterRetentionTrim(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)

	_, err := adapter.InsertVersion(context.Background(), testInsertRequest("demo-rule", "hash-a"), 2)
	require.NoError(t, err)
	_, err = adapter.InsertVersion(context.Background(), testInsertRequest("demo-rule", "hash-b"), 2)
	require.NoError(t, err)

	failingMetaStore := &failOnceStore{ResourceStore: metaStore, failNextUpdate: true, err: errors.New("meta update failed")}
	adapter = NewResourceStoreAdapter(versionStore, intentStore, failingMetaStore)

	intent, err := adapter.CreateIntent(context.Background(), testInsertRequest("demo-rule", "hash-c"))
	require.NoError(t, err)
	require.NoError(t, adapter.MarkIntentApplied(context.Background(), intent.ID))

	_, err = adapter.CommitIntent(context.Background(), intent.ID, 2)
	require.ErrorContains(t, err, "failed to reconcile meta")

	committed, err := adapter.CommitIntent(context.Background(), intent.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, intent.ID, committed.ID)
	assert.Equal(t, int64(3), committed.VersionNo)

	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions, 2)
	assert.Equal(t, int64(3), versions[0].VersionNo)
	assert.Equal(t, int64(2), versions[1].VersionNo)
}

func TestSubscriber_ReconcilesMetaAfterVersionAddedAndMetaFailed(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	failingMetaStore := &failOnceStore{ResourceStore: metaStore, failNextAdd: true, err: errors.New("meta add failed")}
	adapter := NewResourceStoreAdapter(versionStore, intentStore, failingMetaStore)
	sub := NewSubscriber(meshresource.ConditionRouteKind, adapter, 10, locallock.NewLocalLock())
	res := testConditionRule("demo-rule", "v1")

	err := sub.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, res))
	require.ErrorContains(t, err, "failed to reconcile meta")

	err = sub.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, res))
	require.NoError(t, err)

	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions, 1)
	meta, err := adapter.ReconcileMeta(context.Background(), meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.NotNil(t, meta.CurrentVersion)
	assert.Equal(t, versions[0].ID, *meta.CurrentVersion)
}

func TestRecordBootstrap_ReconcilesMetaAfterVersionAddedAndMetaFailed(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	failingMetaStore := &failOnceStore{ResourceStore: metaStore, failNextAdd: true, err: errors.New("meta add failed")}
	adapter := NewResourceStoreAdapter(versionStore, intentStore, failingMetaStore)
	res := testConditionRule("demo-rule", "v1")

	err := recordBootstrapState(context.Background(), adapter, 10, res)
	require.ErrorContains(t, err, "bootstrap version")

	require.NoError(t, recordBootstrapState(context.Background(), adapter, 10, res))
	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, SourceBootstrap, versions[0].Source)
}

func TestResourceStoreAdapter_DuplicateVersionNoIsCorruption(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	key := coremodel.BuildResourceKey("", "demo-rule")
	require.NoError(t, versionStore.Add(testVersionResource("demo-rule", 101, 1, "hash-a", OperationUpdate, SourceAdmin)))
	require.NoError(t, versionStore.Add(testVersionResource("demo-rule", 202, 1, "hash-b", OperationUpdate, SourceAdmin)))

	_, err := adapter.ListVersions(meshresource.ConditionRouteKind, key)
	require.ErrorIs(t, err, ErrVersionLedgerCorrupt)
	assert.Contains(t, err.Error(), "duplicate version number")
	assert.Contains(t, err.Error(), "101")
	assert.Contains(t, err.Error(), "202")
}

func TestResourceStoreAdapter_SharedStoreAdaptersDetectDuplicateVersionNo(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	writerA := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	writerB := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	key := coremodel.BuildResourceKey("", "demo-rule")

	committed, err := writerA.InsertVersion(context.Background(), testInsertRequest("demo-rule", "hash-a"), 10)
	require.NoError(t, err)
	require.NoError(t, versionStore.Add(testVersionResource("demo-rule", 999999, committed.VersionNo, "hash-b", OperationUpdate, SourceAdmin)))

	var reconcileErr error
	require.NotPanics(t, func() {
		_, reconcileErr = writerB.ReconcileMeta(context.Background(), meshresource.ConditionRouteKind, key)
	})
	require.ErrorIs(t, reconcileErr, ErrVersionLedgerCorrupt)
	assert.Contains(t, reconcileErr.Error(), "duplicate version number")
	assert.Contains(t, reconcileErr.Error(), "ConditionRoute")
	assert.Contains(t, reconcileErr.Error(), "demo-rule")
	assert.Contains(t, reconcileErr.Error(), "999999")

	_, err = writerB.InsertVersion(context.Background(), testInsertRequest("demo-rule", "hash-c"), 10)
	require.ErrorIs(t, err, ErrVersionLedgerCorrupt)
}

func TestResourceStoreAdapter_TrimFailureDoesNotFailCommittedVersion(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	failingVersionStore := &failOnceStore{ResourceStore: versionStore, failNextDelete: true, err: errors.New("delete failed")}
	adapter := NewResourceStoreAdapter(failingVersionStore, intentStore, metaStore)

	v1, err := adapter.InsertVersion(context.Background(), testInsertRequest("demo-rule", "hash-a"), 10)
	require.NoError(t, err)
	v2, err := adapter.InsertVersion(context.Background(), testInsertRequest("demo-rule", "hash-b"), 1)
	require.NoError(t, err)
	assert.NotEqual(t, v1.ID, v2.ID)

	meta, err := adapter.ReconcileMeta(context.Background(), meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.NotNil(t, meta.CurrentVersion)
	assert.Equal(t, v2.ID, *meta.CurrentVersion)

	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions, 2)
}

func TestResourceStoreAdapter_CreatedAtAndCommittedAtSemantics(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	req := testInsertRequest("demo-rule", "hash-time")
	req.CreatedAt = time.Unix(100, 0)

	version, err := adapter.InsertVersion(context.Background(), req, 10)
	require.NoError(t, err)
	assert.True(t, version.CreatedAt.Equal(time.Unix(100, 0)))
	assert.True(t, version.CommittedAt.After(version.CreatedAt))

	fixedID := version.ID
	retryReq := req
	retryReq.FixedVersionID = &fixedID
	retried, err := adapter.InsertVersion(context.Background(), retryReq, 10)
	require.NoError(t, err)
	assert.Equal(t, version.ID, retried.ID)
	assert.True(t, retried.CreatedAt.Equal(version.CreatedAt))
	assert.True(t, retried.CommittedAt.Equal(version.CommittedAt))
}

func TestProtoToVersionMissingCommittedAtFallsBackToCreatedAt(t *testing.T) {
	createdAt := time.Unix(100, 0)
	version, err := protoToVersion(&meshproto.RuleVersion{
		ParentRuleKind: string(meshresource.ConditionRouteKind),
		ParentRuleMesh: "",
		ParentRuleName: "demo-rule",
		VersionNo:      1,
		ContentHash:    "hash-a",
		SpecJson:       `{"key":"demo-rule"}`,
		Operation:      string(OperationUpdate),
		Source:         string(SourceAdmin),
		Author:         "admin",
		CreatedAt:      timestamppb.New(createdAt),
	}, 101)
	require.NoError(t, err)
	assert.True(t, version.CreatedAt.Equal(createdAt))
	assert.True(t, version.CommittedAt.Equal(createdAt))
}

func TestResourceStoreAdapter_ListOpenIntentsUsesStatusIndex(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, &noListKeysStore{ResourceStore: intentStore, t: t}, metaStore)
	require.NoError(t, intentStore.Add(testIntentResource("pending", 1, IntentStatusPending, "hash-a")))
	require.NoError(t, intentStore.Add(testIntentResource("applied", 2, IntentStatusApplied, "hash-b")))
	require.NoError(t, intentStore.Add(testIntentResource("failed", 3, IntentStatusFailed, "hash-c")))
	require.NoError(t, intentStore.Add(testIntentResource("committed", 4, IntentStatusCommitted, "hash-d")))

	intents, err := adapter.ListOpenIntents()
	require.NoError(t, err)
	require.Len(t, intents, 2)
	statuses := map[IntentStatus]bool{}
	for _, intent := range intents {
		statuses[intent.Status] = true
	}
	assert.True(t, statuses[IntentStatusPending])
	assert.True(t, statuses[IntentStatusApplied])
}

func TestServiceDisabledMutationsReturnFeatureDisabled(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	svc := NewService(false, 10, adapter)
	res := testConditionRule("demo-rule", "v1")
	req := testInsertRequest("demo-rule", "hash-a")
	intent := &Intent{ID: 1, RuleKind: req.RuleKind, ResourceKey: req.ResourceKey}

	_, err := svc.BeginMutation(context.Background(), res, OperationUpdate, SourceAdmin, "admin", "", nil)
	require.ErrorIs(t, err, ErrFeatureDisabled)
	_, err = svc.RepairIntent(context.Background(), req.RuleKind, req.ResourceKey, res, false)
	require.ErrorIs(t, err, ErrFeatureDisabled)
	_, err = svc.FinalizeMutation(context.Background(), intent, res, false)
	require.ErrorIs(t, err, ErrFeatureDisabled)
	err = svc.AbandonIntent(context.Background(), intent, "operator abort")
	require.ErrorIs(t, err, ErrFeatureDisabled)
}

func TestServiceFinalizeMutationRejectsLeaseLostBeforeCommit(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	svc := NewService(true, 10, adapter)
	res := testConditionRule("demo-rule", "v1")
	req, err := buildMutationInsertRequest(res, OperationUpdate, SourceAdmin, "admin", "", nil, time.Unix(100, 0))
	require.NoError(t, err)
	intent, err := adapter.CreateIntent(context.Background(), req)
	require.NoError(t, err)

	err = corelock.WithLock(context.Background(), locallock.NewLocalLock(), "lease-lost-before-commit", 5*time.Millisecond, func(leaseCtx context.Context) error {
		<-leaseCtx.Done()
		_, err := svc.FinalizeMutation(leaseCtx, intent, res, false)
		return err
	})
	require.ErrorIs(t, err, corelock.ErrLockLeaseLost)

	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Empty(t, versions)

	meta, err := adapter.ReconcileMeta(context.Background(), meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Nil(t, meta)

	open, err := adapter.GetIntent(intent.ID)
	require.NoError(t, err)
	assert.Equal(t, IntentStatusPending, open.Status)
}

func TestServiceFinalizeMutationStopsAfterLeaseLossBeforeLedgerAppend(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	leaseLosingStore := &leaseLosingStore{Store: adapter}
	svc := NewService(true, 10, leaseLosingStore)
	res := testConditionRule("demo-rule", "v1")
	req, err := buildMutationInsertRequest(res, OperationUpdate, SourceAdmin, "admin", "", nil, time.Unix(100, 0))
	require.NoError(t, err)
	intent, err := adapter.CreateIntent(context.Background(), req)
	require.NoError(t, err)

	err = corelock.WithLock(context.Background(), locallock.NewLocalLock(), "lease-lost-before-ledger", 5*time.Millisecond, func(leaseCtx context.Context) error {
		leaseLosingStore.onMarkApplied = func() {
			<-leaseCtx.Done()
		}

		_, err := svc.FinalizeMutation(leaseCtx, intent, res, false)
		return err
	})
	require.ErrorIs(t, err, corelock.ErrLockLeaseLost)

	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Empty(t, versions)

	meta, err := adapter.ReconcileMeta(context.Background(), meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Nil(t, meta)

	open, err := adapter.GetIntent(intent.ID)
	require.NoError(t, err)
	assert.Equal(t, IntentStatusApplied, open.Status)
}

func TestRuleVersionCommitPointRejectsOldLeaseAfterReacquire(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	writerA := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	writerB := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	svcA := NewService(true, 10, writerA)
	lockMgr := locallock.NewLocalLock()
	key := coremodel.BuildResourceKey("", "demo-rule")
	lockKey := corelock.BuildRuleVersioningLockKey(string(meshresource.ConditionRouteKind), "", "demo-rule")

	err := corelock.WithLock(context.Background(), lockMgr, lockKey, 5*time.Millisecond, func(staleLeaseCtx context.Context) error {
		<-staleLeaseCtx.Done()
		err := withRuleVersionLock(lockMgr, meshresource.ConditionRouteKind, key, func(freshLeaseCtx context.Context) error {
			if err := corelock.CheckLease(freshLeaseCtx); err != nil {
				return err
			}
			_, err := writerB.InsertVersion(freshLeaseCtx, testInsertRequest("demo-rule", "fresh-writer"), 10)
			return err
		})
		if err != nil {
			return err
		}

		res := testConditionRule("demo-rule", "stale-writer")
		_, err = svcA.BeginMutation(staleLeaseCtx, res, OperationUpdate, SourceAdmin, "stale", "", nil)
		return err
	})
	require.ErrorIs(t, err, corelock.ErrLockLeaseLost)

	versions, err := writerA.ListVersions(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, "fresh-writer", versions[0].ContentHash)
	open, err := writerA.OpenIntent(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	assert.Nil(t, open)
}

func TestSubscriber_DeleteDedupAndRecreatePreserved(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	sub := NewSubscriber(meshresource.ConditionRouteKind, adapter, 10, locallock.NewLocalLock())
	res := testConditionRule("demo-rule", "A")

	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, res)))
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Deleted, res, nil)))
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Deleted, res, nil)))
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, res)))

	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions, 3)
	assert.Equal(t, []Operation{OperationCreate, OperationDelete, OperationCreate}, []Operation{
		versions[2].Operation,
		versions[1].Operation,
		versions[0].Operation,
	})
}

func TestSubscriber_DoesNotConsumeAdminIntentBySameHash(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	sub := NewSubscriber(meshresource.ConditionRouteKind, adapter, 10, locallock.NewLocalLock())
	res := testConditionRule("demo-rule", "B")
	req, err := buildMutationInsertRequest(res, OperationUpdate, SourceAdmin, "admin", "", nil, time.Unix(100, 0))
	require.NoError(t, err)
	intent, err := adapter.CreateIntent(context.Background(), req)
	require.NoError(t, err)

	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Updated, nil, res)))

	open, err := adapter.GetIntent(intent.ID)
	require.NoError(t, err)
	assert.Equal(t, IntentStatusPending, open.Status)
	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Empty(t, versions)
}

func TestSubscriber_MetadataDoesNotCommitAdminIntent(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	sub := NewSubscriber(meshresource.ConditionRouteKind, adapter, 10, locallock.NewLocalLock())
	res := testConditionRule("demo-rule", "B")
	req, err := buildMutationInsertRequest(res, OperationUpdate, SourceAdmin, "admin", "", nil, time.Unix(100, 0))
	require.NoError(t, err)
	intent, err := adapter.CreateIntent(context.Background(), req)
	require.NoError(t, err)

	event := events.NewResourceChangedEventWithContext(cache.Updated, nil, res, map[string]string{
		"event-source": strconv.FormatInt(intent.ID, 10),
	})
	require.NoError(t, sub.ProcessEvent(event))

	open, err := adapter.GetIntent(intent.ID)
	require.NoError(t, err)
	assert.Equal(t, IntentStatusPending, open.Status)
	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Empty(t, versions)
}

func TestSubscriber_MetadataMismatchStillSkipsOpenIntent(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	sub := NewSubscriber(meshresource.ConditionRouteKind, adapter, 10, locallock.NewLocalLock())
	intentRes := testConditionRule("demo-rule", "B")
	req, err := buildMutationInsertRequest(intentRes, OperationUpdate, SourceAdmin, "admin", "", nil, time.Unix(100, 0))
	require.NoError(t, err)
	intent, err := adapter.CreateIntent(context.Background(), req)
	require.NoError(t, err)

	eventRes := testConditionRule("demo-rule", "C")
	event := events.NewResourceChangedEventWithContext(cache.Updated, nil, eventRes, map[string]string{
		"event-source": strconv.FormatInt(intent.ID, 10),
	})
	err = sub.ProcessEvent(event)
	require.NoError(t, err)

	open, err := adapter.GetIntent(intent.ID)
	require.NoError(t, err)
	assert.Equal(t, IntentStatusPending, open.Status)
	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Empty(t, versions)
}

func TestResourceStoreAdapter_FixedVersionIDConflict(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	fixedID := int64(99)

	req := testInsertRequest("demo-rule", "hash-a")
	req.FixedVersionID = &fixedID
	_, err := adapter.InsertVersion(context.Background(), req, 10)
	require.NoError(t, err)

	conflicting := testInsertRequest("demo-rule", "hash-b")
	conflicting.FixedVersionID = &fixedID
	_, err = adapter.InsertVersion(context.Background(), conflicting, 10)
	require.ErrorIs(t, err, ErrVersionLedgerCorrupt)
}

func TestResourceStoreAdapter_CorruptMetaObjectRejected(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)

	corrupt := meshresource.NewRuleIntentResourceWithAttributes(buildMetaName(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule")), "")
	corrupt.Spec = &meshproto.RuleIntent{Status: string(IntentStatusPending)}
	require.NoError(t, metaStore.Add(corrupt))

	_, err := adapter.InsertVersion(context.Background(), testInsertRequest("demo-rule", "hash-a"), 10)
	require.ErrorIs(t, err, ErrVersionLedgerCorrupt)
}

func TestResourceStoreAdapter_ListVersionsRejectsMalformedVersionName(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)

	corrupt := meshresource.NewRuleVersionResourceWithAttributes("malformed-version-name", "")
	corrupt.Spec = &meshproto.RuleVersion{
		ParentRuleKind: string(meshresource.ConditionRouteKind),
		ParentRuleMesh: "",
		ParentRuleName: "demo-rule",
		VersionNo:      1,
		ContentHash:    "hash-a",
		SpecJson:       `{"key":"demo-rule"}`,
		Operation:      string(OperationUpdate),
		Source:         string(SourceAdmin),
		Author:         "admin",
		CreatedAt:      timestamppb.New(time.Unix(100, 0)),
		CommittedAt:    timestamppb.New(time.Unix(100, 0)),
	}
	require.NoError(t, versionStore.Add(corrupt))

	_, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.ErrorIs(t, err, ErrVersionLedgerCorrupt)
}

func TestResourceStoreAdapter_OpenIntentDuplicateOpenIntents(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)

	require.NoError(t, intentStore.Add(testIntentResource("demo-rule", 1, IntentStatusPending, "hash-a")))
	require.NoError(t, intentStore.Add(testIntentResource("demo-rule", 2, IntentStatusApplied, "hash-b")))

	_, err := adapter.OpenIntent(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.ErrorIs(t, err, ErrVersionLedgerCorrupt)
}

func TestResourceStoreAdapter_OpenIntentDuplicatePendingIntents(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)

	require.NoError(t, intentStore.Add(testIntentResource("demo-rule", 1, IntentStatusPending, "hash-a")))
	require.NoError(t, intentStore.Add(testIntentResource("demo-rule", 2, IntentStatusPending, "hash-b")))

	_, err := adapter.OpenIntent(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.ErrorIs(t, err, ErrVersionLedgerCorrupt)
}

func TestResourceStoreAdapter_CreateIntentRejectsExistingOpenIntent(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)

	first, err := adapter.CreateIntent(context.Background(), testInsertRequest("demo-rule", "hash-a"))
	require.NoError(t, err)

	_, err = adapter.CreateIntent(context.Background(), testInsertRequest("demo-rule", "hash-b"))
	var pending *IntentPendingError
	require.ErrorAs(t, err, &pending)
	assert.Equal(t, first.ID, pending.IntentID)
}

func TestResourceStoreAdapter_CreateIntentSerializesSameParent(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)

	const workers = 16
	var started sync.WaitGroup
	var done sync.WaitGroup
	release := make(chan struct{})
	var successCount int64
	var pendingCount int64
	errCh := make(chan error, workers)

	for i := 0; i < workers; i++ {
		done.Add(1)
		started.Add(1)
		go func(i int) {
			defer done.Done()
			started.Done()
			<-release
			req := testInsertRequest("demo-rule", "hash-"+strconv.Itoa(i))
			_, err := adapter.CreateIntent(context.Background(), req)
			switch err {
			case nil:
				atomic.AddInt64(&successCount, 1)
			default:
				var pending *IntentPendingError
				if errors.As(err, &pending) {
					atomic.AddInt64(&pendingCount, 1)
					return
				}
				errCh <- err
			}
		}(i)
	}
	started.Wait()
	close(release)
	done.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}
	assert.Equal(t, int64(1), successCount)
	assert.Equal(t, int64(workers-1), pendingCount)
	intents, err := adapter.openIntentResources(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Len(t, intents, 1)
}

func TestRuleVersionLockSerializesSharedAdapters(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	writerA := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	writerB := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	lockMgr := locallock.NewLocalLock()
	key := coremodel.BuildResourceKey("", "demo-rule")

	const writes = 24
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
			err := withRuleVersionLock(lockMgr, meshresource.ConditionRouteKind, key, func(context.Context) error {
				_, err := adapter.InsertVersion(context.Background(), testInsertRequest("demo-rule", fmt.Sprintf("hash-%02d", i)), 100)
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
	meta, err := writerB.ReconcileMeta(context.Background(), meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.NotNil(t, meta)
	assert.Equal(t, int64(writes), meta.LastVersionNo)
}

func TestRuleVersionLockSerializesSharedAdapterIntentCreate(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	writerA := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	writerB := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	lockMgr := locallock.NewLocalLock()
	key := coremodel.BuildResourceKey("", "intent-rule")

	const workers = 2
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	var successCount int64
	var pendingCount int64
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			adapter := writerA
			if i == 1 {
				adapter = writerB
			}
			err := withRuleVersionLock(lockMgr, meshresource.ConditionRouteKind, key, func(context.Context) error {
				_, err := adapter.CreateIntent(context.Background(), testInsertRequest("intent-rule", fmt.Sprintf("hash-%d", i)))
				return err
			})
			if err == nil {
				atomic.AddInt64(&successCount, 1)
				return
			}
			var pending *IntentPendingError
			if errors.As(err, &pending) {
				atomic.AddInt64(&pendingCount, 1)
				return
			}
			errCh <- err
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	assert.Equal(t, int64(1), successCount)
	assert.Equal(t, int64(1), pendingCount)
	intents, err := writerA.ListOpenIntents()
	require.NoError(t, err)
	require.Len(t, intents, 1)
	assert.Equal(t, "intent-rule", intents[0].RuleName)
}

func TestRecordBootstrapLockedSharedAdaptersSingleBaseline(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	writerA := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	writerB := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	lockMgr := locallock.NewLocalLock()
	res := testConditionRule("bootstrap-rule", "v1")
	rm := &singleResourceManager{res: res}

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	for _, writer := range []*ResourceStoreAdapter{writerA, writerB} {
		wg.Add(1)
		go func(writer *ResourceStoreAdapter) {
			defer wg.Done()
			errCh <- RecordBootstrapLocked(writer, 10, res.ResourceKind(), res.ResourceKey(), rm, lockMgr)
		}(writer)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	versions, err := writerA.ListVersions(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, SourceBootstrap, versions[0].Source)
	assert.Equal(t, int64(1), versions[0].VersionNo)
}

func TestRecordBootstrapLockedAndSubscriberEventNoDuplicate(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	bootstrapWriter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	eventWriter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	lockMgr := locallock.NewLocalLock()
	res := testConditionRule("bootstrap-event-rule", "v1")
	rm := &singleResourceManager{res: res}
	sub := NewSubscriber(meshresource.ConditionRouteKind, eventWriter, 10, lockMgr)

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		errCh <- RecordBootstrapLocked(bootstrapWriter, 10, res.ResourceKind(), res.ResourceKey(), rm, lockMgr)
	}()
	go func() {
		defer wg.Done()
		errCh <- sub.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, res))
	}()
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	versions, err := bootstrapWriter.ListVersions(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, int64(1), versions[0].VersionNo)
}

func TestRuleVersionLockAllowsDifferentParentsInParallel(t *testing.T) {
	lockMgr := locallock.NewLocalLock()
	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondEntered := make(chan struct{})

	go func() {
		err := withRuleVersionLock(lockMgr, meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "rule-a"), func(context.Context) error {
			close(firstEntered)
			<-releaseFirst
			return nil
		})
		require.NoError(t, err)
	}()
	<-firstEntered

	err := withRuleVersionLock(lockMgr, meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "rule-b"), func(context.Context) error {
		close(secondEntered)
		return nil
	})
	require.NoError(t, err)

	select {
	case <-secondEntered:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("different parent rules should not be blocked by a global lock")
	}
	close(releaseFirst)
}

func TestResourceStoreAdapter_RetriesVersionIDCollision(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	adapter.idGenerator = &sequenceIDGenerator{ids: []int64{7, 8}}
	require.NoError(t, versionStore.Add(testVersionResource("demo-rule", 7, 1, "existing", OperationUpdate, SourceAdmin)))

	version, err := adapter.InsertVersion(context.Background(), testInsertRequest("demo-rule", "new-hash"), 10)
	require.NoError(t, err)
	assert.Equal(t, int64(8), version.ID)
	assert.Equal(t, int64(2), version.VersionNo)
}

func TestResourceStoreAdapter_RetriesIntentIDCollision(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	adapter.idGenerator = &sequenceIDGenerator{ids: []int64{7, 8}}
	require.NoError(t, intentStore.Add(testIntentResource("demo-rule", 7, IntentStatusFailed, "old-hash")))

	intent, err := adapter.CreateIntent(context.Background(), testInsertRequest("demo-rule", "new-hash"))
	require.NoError(t, err)
	assert.Equal(t, int64(8), intent.ID)
}

func TestService_RepairPendingIntentRequiresCurrentResourceMatch(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	svc := NewService(true, 10, adapter)
	res := testConditionRule("demo-rule", "v1")
	req, err := buildMutationInsertRequest(res, OperationUpdate, SourceAdmin, "admin", "", nil, time.Unix(100, 0))
	require.NoError(t, err)
	intent, err := adapter.CreateIntent(context.Background(), req)
	require.NoError(t, err)

	mismatch := testConditionRule("demo-rule", "v2")
	_, err = finalizeMutationForTest(svc, intent, mismatch, false)
	var pending *IntentPendingError
	require.ErrorAs(t, err, &pending)

	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Empty(t, versions)

	repaired, err := finalizeMutationForTest(svc, intent, res, false)
	require.NoError(t, err)
	require.NotNil(t, repaired)
	assert.Equal(t, intent.ID, repaired.IntentID)
}

func TestService_RepairAppliedIntentRejectsOutcomeMismatch(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	svc := NewService(true, 10, adapter)
	res := testConditionRule("demo-rule", "v1")
	req, err := buildMutationInsertRequest(res, OperationUpdate, SourceAdmin, "admin", "", nil, time.Unix(100, 0))
	require.NoError(t, err)
	intent, err := adapter.CreateIntent(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, adapter.MarkIntentApplied(context.Background(), intent.ID))

	mismatch := testConditionRule("demo-rule", "v2")
	_, err = finalizeMutationForTest(svc, intent, mismatch, false)
	require.ErrorIs(t, err, ErrIntentOutcomeMismatch)

	versions, err := adapter.ListVersions(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", "demo-rule"))
	require.NoError(t, err)
	require.Empty(t, versions)
}

func TestResourceStoreAdapter_CheckExpectedVersion(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	key := coremodel.BuildResourceKey("", "demo-rule")

	require.NoError(t, adapter.CheckExpectedVersion(meshresource.ConditionRouteKind, key, nil))

	expected := int64(1)
	err := adapter.CheckExpectedVersion(meshresource.ConditionRouteKind, key, &expected)
	var conflict *ConflictError
	require.ErrorAs(t, err, &conflict)
	require.Nil(t, conflict.CurrentVersionID)

	expectedDeleted := int64(0)
	require.NoError(t, adapter.CheckExpectedVersion(meshresource.ConditionRouteKind, key, &expectedDeleted))

	version, err := adapter.InsertVersion(context.Background(), testInsertRequest("demo-rule", "hash-a"), 10)
	require.NoError(t, err)
	err = adapter.CheckExpectedVersion(meshresource.ConditionRouteKind, key, &expectedDeleted)
	require.ErrorAs(t, err, &conflict)
	require.NotNil(t, conflict.CurrentVersionID)

	mismatch := version.ID + 1
	err = adapter.CheckExpectedVersion(meshresource.ConditionRouteKind, key, &mismatch)
	require.ErrorAs(t, err, &conflict)
	require.NotNil(t, conflict.CurrentVersionID)
	assert.Equal(t, version.ID, *conflict.CurrentVersionID)

	require.NoError(t, adapter.CheckExpectedVersion(meshresource.ConditionRouteKind, key, &version.ID))
}

func TestService_DiffAgainstCurrentPreviousAndExplicitVersion(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	svc := NewService(true, 10, adapter)

	v1, err := adapter.InsertVersion(context.Background(), testInsertRequest("demo-rule", "hash-a"), 10)
	require.NoError(t, err)
	v2, err := adapter.InsertVersion(context.Background(), testInsertRequest("demo-rule", "hash-b"), 10)
	require.NoError(t, err)

	diff, err := svc.Diff(meshresource.ConditionRouteKind, "", "demo-rule", v1.ID, "current")
	require.NoError(t, err)
	assert.Equal(t, v1.ID, diff.Left.ID)
	assert.Equal(t, v2.ID, diff.Right.ID)

	diff, err = svc.Diff(meshresource.ConditionRouteKind, "", "demo-rule", v2.ID, "previous")
	require.NoError(t, err)
	assert.Equal(t, v2.ID, diff.Left.ID)
	assert.Equal(t, v1.ID, diff.Right.ID)

	diff, err = svc.Diff(meshresource.ConditionRouteKind, "", "demo-rule", v1.ID, formatTestID(v2.ID))
	require.NoError(t, err)
	assert.Equal(t, v2.ID, diff.Right.ID)

	_, err = svc.Diff(meshresource.ConditionRouteKind, "", "demo-rule", v1.ID, "not-a-version")
	var bizErr bizerror.Error
	require.ErrorAs(t, err, &bizErr)

	_, err = svc.Diff(meshresource.ConditionRouteKind, "", "demo-rule", v1.ID, "previous")
	require.ErrorIs(t, err, ErrVersionNotFound)
}

func TestService_DiffAgainstCurrentUsesDeleteMarkerWhenDeleted(t *testing.T) {
	versionStore, intentStore, metaStore := newVersioningStores(t)
	adapter := NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	svc := NewService(true, 10, adapter)

	v1, err := adapter.InsertVersion(context.Background(), testInsertRequest("demo-rule", "hash-a"), 10)
	require.NoError(t, err)
	deleteReq := testInsertRequest("demo-rule", HashSpecJSON(DeleteSpecJSON))
	deleteReq.Operation = OperationDelete
	deleteReq.SpecJSON = DeleteSpecJSON
	deleteReq.ContentHash = HashSpecJSON(DeleteSpecJSON)
	deleted, err := adapter.InsertVersion(context.Background(), deleteReq, 10)
	require.NoError(t, err)

	diff, err := svc.Diff(meshresource.ConditionRouteKind, "", "demo-rule", v1.ID, "current")
	require.NoError(t, err)
	assert.Equal(t, v1.ID, diff.Left.ID)
	assert.Equal(t, deleted.ID, diff.Right.ID)
	assert.Equal(t, DeleteSpecJSON, diff.Right.SpecJSON)
}

func newVersioningStores(t *testing.T) (store.ResourceStore, store.ResourceStore, store.ResourceStore) {
	t.Helper()
	versionStore := memoryst.NewMemoryResourceStore(meshresource.RuleVersionKind)
	intentStore := memoryst.NewMemoryResourceStore(meshresource.RuleIntentKind)
	metaStore := memoryst.NewMemoryResourceStore(meshresource.RuleMetaKind)
	for _, s := range []store.ManagedResourceStore{versionStore, intentStore, metaStore} {
		require.NoError(t, s.Init(nil))
	}
	return versionStore, intentStore, metaStore
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

func testIntentResource(ruleName string, id int64, status IntentStatus, hash string) *meshresource.RuleIntentResource {
	res := meshresource.NewRuleIntentResourceWithAttributes(buildIntentName(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", ruleName), id), "")
	res.Spec = &meshproto.RuleIntent{
		ParentRuleKind: string(meshresource.ConditionRouteKind),
		ParentRuleMesh: "",
		ParentRuleName: ruleName,
		ContentHash:    hash,
		SpecJson:       `{"key":"` + ruleName + `","hash":"` + hash + `"}`,
		Operation:      string(OperationUpdate),
		Source:         string(SourceAdmin),
		Author:         "admin",
		Status:         string(status),
		CreatedAt:      timestamppb.New(time.Unix(100, 0)),
	}
	return res
}

func testVersionResource(ruleName string, id, versionNo int64, hash string, op Operation, source Source) *meshresource.RuleVersionResource {
	res := meshresource.NewRuleVersionResourceWithAttributes(buildVersionName(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", ruleName), id), "")
	res.Spec = &meshproto.RuleVersion{
		ParentRuleKind: string(meshresource.ConditionRouteKind),
		ParentRuleMesh: "",
		ParentRuleName: ruleName,
		VersionNo:      versionNo,
		ContentHash:    hash,
		SpecJson:       fmt.Sprintf(`{"key":%q,"hash":%q}`, ruleName, hash),
		Operation:      string(op),
		Source:         string(source),
		Author:         "admin",
		CreatedAt:      timestamppb.New(time.Unix(100, 0)),
		CommittedAt:    timestamppb.New(time.Unix(101, 0)),
	}
	return res
}

func testConditionRule(ruleName, payload string) *meshresource.ConditionRouteResource {
	res := meshresource.NewConditionRouteResourceWithAttributes(ruleName, "")
	res.Spec = &meshproto.ConditionRoute{Enabled: true, Key: ruleName, Conditions: []string{payload}}
	return res
}

func finalizeMutationForTest(svc *Service, intent *Intent, current coremodel.Resource, deleted bool) (*Version, error) {
	var version *Version
	err := withRuleVersionLock(locallock.NewLocalLock(), intent.RuleKind, intent.ResourceKey, func(leaseCtx context.Context) error {
		var inner error
		version, inner = svc.FinalizeMutation(leaseCtx, intent, current, deleted)
		return inner
	})
	return version, err
}

func formatTestID(id int64) string {
	return strconv.FormatInt(id, 10)
}

type noListKeysStore struct {
	store.ResourceStore
	t *testing.T
}

func (s *noListKeysStore) ListKeys() []string {
	s.t.Fatalf("GetIntent must use the RuleIntent ID index instead of ListKeys")
	return nil
}

type singleResourceManager struct {
	res coremodel.Resource
}

func (rm *singleResourceManager) GetByKey(kind coremodel.ResourceKind, key string) (coremodel.Resource, bool, error) {
	if rm.res != nil && rm.res.ResourceKind() == kind && rm.res.ResourceKey() == key {
		return rm.res, true, nil
	}
	return nil, false, nil
}

func (rm *singleResourceManager) GetByKeys(coremodel.ResourceKind, []string) ([]coremodel.Resource, error) {
	return nil, nil
}

func (rm *singleResourceManager) ListByIndexes(coremodel.ResourceKind, []index.IndexCondition) ([]coremodel.Resource, error) {
	return nil, nil
}

func (rm *singleResourceManager) PageListByIndexes(coremodel.ResourceKind, []index.IndexCondition, coremodel.PageReq) (*coremodel.PageData[coremodel.Resource], error) {
	return nil, nil
}

func (rm *singleResourceManager) GetStore(coremodel.ResourceKind) (store.ResourceStore, error) {
	return nil, nil
}

func (rm *singleResourceManager) Add(context.Context, coremodel.Resource) error {
	return nil
}

func (rm *singleResourceManager) Update(context.Context, coremodel.Resource) error {
	return nil
}

func (rm *singleResourceManager) Upsert(context.Context, coremodel.Resource) error {
	return nil
}

func (rm *singleResourceManager) DeleteByKey(context.Context, coremodel.ResourceKind, string, string) error {
	return nil
}

var _ manager.ResourceManager = (*singleResourceManager)(nil)

type failOnceStore struct {
	store.ResourceStore
	failNextAdd    bool
	failNextUpdate bool
	failNextDelete bool
	err            error
}

func (s *failOnceStore) Add(obj interface{}) error {
	if s.failNextAdd {
		s.failNextAdd = false
		return s.err
	}
	return s.ResourceStore.Add(obj)
}

func (s *failOnceStore) Update(obj interface{}) error {
	if s.failNextUpdate {
		s.failNextUpdate = false
		return s.err
	}
	return s.ResourceStore.Update(obj)
}

func (s *failOnceStore) Delete(obj interface{}) error {
	if s.failNextDelete {
		s.failNextDelete = false
		return s.err
	}
	return s.ResourceStore.Delete(obj)
}

type leaseLosingStore struct {
	Store
	onMarkApplied func()
}

func (s *leaseLosingStore) MarkIntentApplied(ctx context.Context, id int64) error {
	err := s.Store.MarkIntentApplied(ctx, id)
	if err == nil && s.onMarkApplied != nil {
		s.onMarkApplied()
	}
	return err
}

type sequenceIDGenerator struct {
	mu  sync.Mutex
	ids []int64
}

func (g *sequenceIDGenerator) Next() (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.ids) == 0 {
		return 0, errors.New("no test ids left")
	}
	id := g.ids[0]
	g.ids = g.ids[1:]
	return id, nil
}
