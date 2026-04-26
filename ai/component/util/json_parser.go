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
 * WITHOUT WARRANTIES OR CONDITIONS OR ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"dubbo-admin-ai/runtime"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// ParseOutputWithRetry 尝试解析 LLM 输出，如果失败则使用 LLM 修正格式
func ParseOutputWithRetry[T any](
	ctx context.Context,
	rawText string,
	resp *ai.ModelResponse,
	targetType T,
	modelName string,
) (T, error) {
	var result T
	var zero T

	// 第一次尝试：直接解析
	parseErr := tryParse(rawText, &result)
	if parseErr == nil {
		return result, nil
	}

	// 解析失败，记录日志
	runtime.GetLogger().Warn("Failed to parse LLM output, attempting auto-fix",
		"error", parseErr,
		"raw_text", truncate(rawText, 500),
	)

	// 第二次尝试：提取 JSON 代码块
	extracted := extractJSONFromCodeBlock(rawText)
	if extracted != "" {
		var extractErr error
		extractErr = tryParse(extracted, &result)
		if extractErr == nil {
			runtime.GetLogger().Info("Successfully extracted JSON from code block")
			return result, nil
		}
		// 提取后仍失败，更新 parseErr
		parseErr = fmt.Errorf("extracted JSON also invalid: %w", extractErr)
	}

	// 第三次尝试：使用 LLM 修正格式
	fixed, llmErr := fixJSONWithLLM(ctx, rawText, modelName)
	if llmErr != nil {
		return zero, fmt.Errorf("failed to fix JSON with LLM: %w, original parse error: %w", llmErr, parseErr)
	}

	// 解析修正后的结果
	if err := tryParse(fixed, &result); err != nil {
		return zero, fmt.Errorf("failed to parse fixed JSON: %w, fixed: %s", err, truncate(fixed, 500))
	}

	runtime.GetLogger().Info("Successfully fixed JSON with LLM")
	return result, nil
}

// truncate 截断字符串到指定长度
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// tryParse 尝试解析 JSON
func tryParse(text string, target any) error {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

// extractJSONFromCodeBlock 从 markdown 代码块中提取 JSON
func extractJSONFromCodeBlock(text string) string {
	// 匹配 ```json ... ``` 或 ``` ... ```
	patterns := []string{
		"```json\\s*([\\s\\S]*?)\\s*```",
		"```\\s*([\\s\\S]*?)\\s*```",
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	// 尝试找到第一个 { 和最后一个 }
	firstBrace := strings.Index(text, "{")
	lastBrace := strings.LastIndex(text, "}")
	if firstBrace >= 0 && lastBrace > firstBrace {
		return text[firstBrace : lastBrace+1]
	}

	return ""
}

// fixJSONWithLLM 使用 LLM 修正 JSON 格式
func fixJSONWithLLM(ctx context.Context, invalidJSON string, modelName string) (string, error) {
	// 直接从全局 runtime 获取，无需修改 runtime.go
	rt := runtime.GetRuntime()
	g := rt.GetGenkitRegistry()
	if g == nil {
		return "", fmt.Errorf("genkit registry not initialized")
	}

	promptText := fmt.Sprintf(`You are a JSON formatter. Convert the following text into valid JSON output.
Rules:
- Return ONLY the raw JSON object, no markdown, no code blocks, no explanation
- Extract the JSON content if it's wrapped in markdown or other text
- Fix any syntax errors while preserving the original meaning and structure
- Start with { and end with }

Text to fix:
%s

Valid JSON output:`, invalidJSON)

	prompt := genkit.DefinePrompt(g, "json-fixer",
		ai.WithSystem("You are a JSON formatter. Output only valid JSON."),
		ai.WithModelName(modelName),
	)

	resp, err := prompt.Execute(ctx, ai.WithMessages(ai.NewMessage(ai.RoleUser, nil, ai.NewTextPart(promptText))))
	if err != nil {
		return "", fmt.Errorf("failed to execute JSON fixer prompt: %w", err)
	}

	if resp == nil || resp.Text() == "" {
		return "", fmt.Errorf("empty response from JSON fixer")
	}

	fixed := strings.TrimSpace(resp.Text())
	runtime.GetLogger().Debug("LLM fixed JSON", "fixed", fixed)

	return fixed, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
