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

	"dubbo-admin-ai/component/hooks"
	"github.com/firebase/genkit/go/ai"
)

// measureContextSize observes the messages already selected for one model call;
// it must not query the Store or change the interaction's stable snapshot.
func measureContextSize(messages []*ai.Message) hooks.ContextSize {
	size := hooks.ContextSize{Messages: len(messages)}
	if len(messages) == 0 {
		return size
	}
	encoded, err := json.Marshal(messages)
	if err != nil {
		size.JSONBytes = -1
		size.EstimatedTokens = -1
		return size
	}
	size.JSONBytes = len(encoded)
	// Four JSON bytes per token is deliberately a rough, model-independent
	// observation. It is not an upper bound, especially for non-English text.
	size.EstimatedTokens = (size.JSONBytes + 3) / 4
	return size
}
