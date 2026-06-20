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

	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// Store defines the storage interface for rule versioning.
// Production implementation: ResourceStoreAdapter (uses resource store)
type Store interface {
	InsertVersion(ctx context.Context, req InsertRequest, maxVersions int64) (*Version, error)
	CreateIntent(ctx context.Context, req InsertRequest) (*Intent, error)
	GetIntent(id int64) (*Intent, error)
	OpenIntent(kind coremodel.ResourceKind, resourceKey string) (*Intent, error)
	MarkIntentApplied(ctx context.Context, id int64) error
	MarkIntentOutcomeUnknown(ctx context.Context, id int64, message string) error
	MarkIntentObserved(ctx context.Context, id int64, op Operation, contentHash, specJSON string) error
	MarkIntentFailed(ctx context.Context, id int64, message string) error
	CommitIntent(ctx context.Context, id int64, maxVersions int64) (*Version, error)
	CleanupIntent(id int64, terminalStatus IntentStatus) error
	ListOpenIntents() ([]Intent, error)
	ListTerminalIntents() ([]Intent, error)
	ListVersions(kind coremodel.ResourceKind, resourceKey string) ([]Version, error)
	LedgerSnapshot(kind coremodel.ResourceKind, resourceKey string) (*LedgerSnapshot, error)
	GetVersion(kind coremodel.ResourceKind, resourceKey string, id int64) (*Version, error)
	LatestVersion(kind coremodel.ResourceKind, resourceKey string) (*Version, error)
	CheckExpectedVersion(kind coremodel.ResourceKind, resourceKey string, expected *int64) error
}
