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

package react

import (
	"context"
	"encoding/json"
	"fmt"

	"dubbo-admin-ai/schema"

	"github.com/firebase/genkit/go/ai"
)

type currentPageContextKey struct{}

type currentPageContextEnvelope struct {
	Kind    string                    `json:"kind"`
	Trust   string                    `json:"trust"`
	Context *schema.AIContextSnapshot `json:"context"`
}

func withCurrentPageContext(ctx context.Context, snapshot *schema.AIContextSnapshot) context.Context {
	if snapshot == nil {
		return ctx
	}
	return context.WithValue(ctx, currentPageContextKey{}, snapshot)
}

func injectCurrentPageContext(ctx context.Context, messages []*ai.Message) ([]*ai.Message, error) {
	snapshot, ok := ctx.Value(currentPageContextKey{}).(*schema.AIContextSnapshot)
	if !ok || snapshot == nil {
		return messages, nil
	}

	payload, err := json.Marshal(currentPageContextEnvelope{
		Kind:    "page_context",
		Trust:   "untrusted_observation",
		Context: snapshot,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal current AI context: %w", err)
	}

	insertAt := len(messages)
	for index := len(messages) - 1; index >= 0; index-- {
		if messages[index].Role == ai.RoleUser {
			insertAt = index
			break
		}
	}

	result := make([]*ai.Message, 0, len(messages)+1)
	result = append(result, messages[:insertAt]...)
	result = append(result, ai.NewUserMessage(ai.NewJSONPart(string(payload))))
	result = append(result, messages[insertAt:]...)
	return result, nil
}
