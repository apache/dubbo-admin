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
	"encoding/json"
	"testing"

	"github.com/firebase/genkit/go/ai"
)

func TestContextSizeIncludesStructuredToolDataAndUTF8Bytes(t *testing.T) {
	messages := []*ai.Message{
		ai.NewUserTextMessage("解释这个配置"),
		ai.NewMessage(ai.RoleModel, nil, ai.NewToolResponsePart(&ai.ToolResponse{
			Name: "lookup", Ref: "call-1", Output: map[string]any{"result": "工具返回的证据"},
		})),
	}
	before, err := json.Marshal(messages)
	if err != nil {
		t.Fatal(err)
	}
	size := measureContextSize(messages)
	if size.Messages != 2 || size.JSONBytes != len(before) || size.EstimatedTokens != (len(before)+3)/4 {
		t.Fatalf("context size = %+v, JSON bytes = %d", size, len(before))
	}
	textOnly := measureContextSize(messages[:1])
	if size.JSONBytes <= textOnly.JSONBytes {
		t.Fatal("structured tool payload was not counted")
	}
	after, err := json.Marshal(messages)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("measurement mutated canonical messages")
	}
}

func TestContextSizeEncodingFailureLeavesCountAvailable(t *testing.T) {
	message := ai.NewUserTextMessage("hello")
	message.Metadata = map[string]any{"unsupported": func() {}}
	size := measureContextSize([]*ai.Message{message})
	if size.Messages != 1 || size.JSONBytes != -1 || size.EstimatedTokens != -1 {
		t.Fatalf("failed encoding size = %+v, want count with unavailable byte/token values", size)
	}
}
