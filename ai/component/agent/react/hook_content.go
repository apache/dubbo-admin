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

import "github.com/firebase/genkit/go/ai"

type genAIMessage struct {
	Role         string             `json:"role"`
	Parts        []genAIMessagePart `json:"parts"`
	FinishReason string             `json:"finish_reason,omitempty"`
}

type genAIMessagePart struct {
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments any    `json:"arguments,omitempty"`
	Result    any    `json:"result,omitempty"`
}

func genAIInputMessages(input any) any {
	messages, ok := input.([]*ai.Message)
	if !ok {
		return nil
	}
	return convertGenAIMessages(messages, "")
}

func genAIOutputMessages(output any) any {
	response, ok := output.(*ai.ModelResponse)
	if !ok || response == nil || response.Message == nil {
		return nil
	}
	return convertGenAIMessages([]*ai.Message{response.Message}, string(response.FinishReason))
}

func convertGenAIMessages(messages []*ai.Message, finishReason string) []genAIMessage {
	converted := make([]genAIMessage, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			continue
		}
		parts := make([]genAIMessagePart, 0, len(message.Content))
		for _, part := range message.Content {
			switch {
			case part == nil:
				continue
			case part.IsText():
				parts = append(parts, genAIMessagePart{Type: "text", Content: part.Text})
			case part.IsToolRequest() && part.ToolRequest != nil:
				parts = append(parts, genAIMessagePart{
					Type:      "tool_call",
					ID:        part.ToolRequest.Ref,
					Name:      part.ToolRequest.Name,
					Arguments: part.ToolRequest.Input,
				})
			case part.IsToolResponse() && part.ToolResponse != nil:
				parts = append(parts, genAIMessagePart{
					Type:   "tool_call_response",
					ID:     part.ToolResponse.Ref,
					Result: part.ToolResponse.Output,
				})
			}
		}
		role := string(message.Role)
		if message.Role == ai.RoleModel {
			role = "assistant"
		}
		converted = append(converted, genAIMessage{Role: role, Parts: parts, FinishReason: finishReason})
	}
	return converted
}
