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
	"time"

	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// RecordBootstrapState creates a baseline history entry for a rule that already
// exists before rule history starts tracking it. Bootstrap is not reconciliation:
// once any history exists, startup must not record the current registry state as
// an UPDATE.
func RecordBootstrapState(ctx context.Context, store *ResourceStoreAdapter, maxVersions int64, res coremodel.Resource) error {
	if store == nil {
		return ErrVersionStoreError
	}
	if res == nil {
		return nil
	}
	latest, err := store.LatestVersion(res.ResourceKind(), res.ResourceKey())
	if err != nil && !errors.Is(err, ErrVersionNotFound) {
		return err
	}
	if latest != nil {
		return nil
	}

	hash, specJSON, err := NormalizeResource(res)
	if err != nil {
		return err
	}
	req := InsertRequest{
		RuleKind:    res.ResourceKind(),
		Mesh:        res.ResourceMesh(),
		ResourceKey: res.ResourceKey(),
		RuleName:    res.ResourceMeta().Name,
		SpecJSON:    specJSON,
		ContentHash: hash,
		Source:      SourceBootstrap,
		Operation:   OperationCreate,
		Author:      "system:bootstrap",
		Reason:      "initial rule history baseline",
		CreatedAt:   time.Now(),
	}
	if _, err := store.InsertVersion(ctx, req, maxVersions); err != nil {
		return fmt.Errorf("bootstrap version for %s failed: %w", res.ResourceKey(), err)
	}
	return nil
}
