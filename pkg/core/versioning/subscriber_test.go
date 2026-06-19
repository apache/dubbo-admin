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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/core/events"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func TestNormalizeRuleEvent(t *testing.T) {
	res := testConditionRule("demo-rule", "v1")
	hash, specJSON, err := NormalizeResource(res)
	require.NoError(t, err)

	tests := []struct {
		name     string
		event    events.Event
		wantOp   Operation
		wantHash string
		wantSpec string
		wantNil  bool
	}{
		{
			name:     "create uses new resource",
			event:    events.NewResourceChangedEvent(cache.Added, nil, res),
			wantOp:   OperationCreate,
			wantHash: hash,
			wantSpec: specJSON,
		},
		{
			name:     "update uses new resource",
			event:    events.NewResourceChangedEvent(cache.Updated, testConditionRule("demo-rule", "old"), res),
			wantOp:   OperationUpdate,
			wantHash: hash,
			wantSpec: specJSON,
		},
		{
			name:     "delete uses old resource and delete marker",
			event:    events.NewResourceChangedEvent(cache.Deleted, res, nil),
			wantOp:   OperationDelete,
			wantHash: HashSpecJSON(DeleteSpecJSON),
			wantSpec: DeleteSpecJSON,
		},
		{
			name:    "unknown event type ignored",
			event:   events.NewResourceChangedEvent(cache.Sync, nil, res),
			wantNil: true,
		},
		{
			name:    "nil selected resource ignored",
			event:   events.NewResourceChangedEvent(cache.Added, nil, nil),
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeRuleEvent(tt.event)
			require.NoError(t, err)
			if tt.wantNil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, meshresource.ConditionRouteKind, got.Parent.Kind)
			assert.Equal(t, "demo-rule", got.Parent.Name)
			assert.Equal(t, res.ResourceKey(), got.Parent.ResourceKey)
			assert.Equal(t, tt.wantOp, got.Operation)
			assert.Equal(t, tt.wantHash, got.ContentHash)
			assert.Equal(t, []byte(tt.wantSpec), got.SpecJSON)
		})
	}
}

func TestNormalizeRuleEventParsesIntentTokenOnce(t *testing.T) {
	res := testConditionRule("demo-rule", "v1")
	event := events.NewResourceChangedEventWithContext(cache.Updated, nil, res, map[string]string{
		IntentIDEventContextKey: "42",
	})

	normalized, err := normalizeRuleEvent(event)
	require.NoError(t, err)
	require.NotNil(t, normalized)
	assert.Equal(t, int64(42), normalized.MutationIntentID)
}

func TestNormalizeRuleEventRejectsInvalidIntentToken(t *testing.T) {
	res := testConditionRule("demo-rule", "v1")
	event := events.NewResourceChangedEventWithContext(cache.Updated, nil, res, map[string]string{
		IntentIDEventContextKey: "not-an-int",
	})

	_, err := normalizeRuleEvent(event)
	require.ErrorIs(t, err, ErrVersionLedgerCorrupt)
}
