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
	"strings"
	"testing"

	"github.com/firebase/genkit/go/ai"
)

func TestModelInputViewSanitizesCopyWithoutChangingCanonicalMessage(t *testing.T) {
	const secret = "model-view-secret"
	canonical := ai.NewMessage(ai.RoleUser, map[string]any{"password": secret},
		ai.NewTextPart("Authorization: Bearer "+secret),
		ai.NewToolRequestPart(&ai.ToolRequest{Name: "lookup", Input: map[string]any{"api_key": secret}}),
	)
	view, err := modelInputView([]*ai.Message{canonical})
	if err != nil {
		t.Fatal(err)
	}
	if len(view) != 1 || view[0] == canonical {
		t.Fatalf("modelInputView() did not create a copy: %#v", view)
	}
	if strings.Contains(view[0].Content[0].Text, secret) {
		t.Fatalf("model view leaked secret: %#v", view[0])
	}
	if canonical.Content[0].Text != "Authorization: Bearer "+secret || canonical.Metadata["password"] != secret {
		t.Fatalf("canonical message was changed: %#v", canonical)
	}
}
