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

package gormstore

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/firebase/genkit/go/ai"
)

func TestMessageCodecRoundTrip(t *testing.T) {
	original := ai.NewMessage(ai.RoleModel, nil,
		ai.NewTextPart("answer"),
		ai.NewTextPart("with context"),
	)

	payload, err := encodeMessage(original)
	if err != nil {
		t.Fatalf("encodeMessage() error = %v", err)
	}
	decoded, err := decodeMessage(42, payload)
	if err != nil {
		t.Fatalf("decodeMessage() error = %v", err)
	}
	originalJSON, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal original message error = %v", err)
	}
	decodedJSON, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("marshal decoded message error = %v", err)
	}
	if !bytes.Equal(decodedJSON, originalJSON) {
		t.Fatalf("decoded message JSON = %s, want %s", decodedJSON, originalJSON)
	}
}

func TestMessageCodecRejectsNilMessage(t *testing.T) {
	if _, err := encodeMessage(nil); err == nil {
		t.Fatal("encodeMessage(nil) succeeded, want error")
	}
}
