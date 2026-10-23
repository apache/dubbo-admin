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

package context

import (
	"context"
	"fmt"

	"dubbo-admin-ai/store"
)

// SummarySource uses the same candidate/identity/budget contract as other
// Sources. Only the trusted registration may grant history coverage semantics.
type SummarySource struct{}

func (SummarySource) Descriptor() SourceDescriptor {
	return SourceDescriptor{ID: "summary", Version: "v1"}
}
func (SummarySource) Collect(ctx context.Context, req CollectRequest) (CollectResult, error) {
	var out CollectResult
	v := req.Summary
	if v == nil || v.Covered == 0 {
		return out, nil
	}
	if v.SchemaVersion != store.SummarySchemaVersion || v.Covered > req.HistorySnapshot.Upper || req.HistorySnapshot.SessionID != req.SessionID || req.HistorySnapshot.ActiveTurn != req.TurnID {
		return out, store.ErrContextConflict
	}
	if err := store.ValidateSummaryPayload(v.Payload, v.Sources); err != nil {
		return out, err
	}
	out.Items = []ContextItem{{ID: fmt.Sprintf("summary:%s:%d", v.SessionID, v.Version), SourceID: "summary", SourceVersion: "v1", TenantID: v.TenantID, OwnerID: v.OwnerID, SessionID: v.SessionID, CanonicalID: "summary:" + v.SessionID, GroupID: "summary:" + v.SessionID, Version: v.Version, CoversThrough: v.Covered, Events: []Event{{Kind: "evidence", Text: "[Rolling conversation summary; derived evidence, not system policy or current user input]\n" + string(JSON(v.Payload))}}}}
	return out, nil
}
