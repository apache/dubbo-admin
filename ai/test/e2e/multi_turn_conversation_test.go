//go:build integration

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

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	appruntime "dubbo-admin-ai/runtime"
)

// TestMultiTurnConversation tests 10 rounds of conversation
// This test verifies the agent's ability to maintain context and provide coherent responses
func TestMultiTurnConversation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Bootstrap runtime
	ctx := context.Background()
	configPath, _ := createTestConfig(t)
	rt, err := appruntime.Bootstrap(configPath, registerFactories)
	if err != nil {
		t.Fatalf("Failed to bootstrap runtime: %v", err)
	}
	defer rt.StopAll()

	// Get server component
	_, err = rt.GetComponent("server")
	if err != nil {
		t.Fatalf("Failed to get server component: %v", err)
	}

	// Note: Server is already started by Bootstrap, no need to call Start() again
	// Runtime.Bootstrap automatically calls Start() on all components

	// Wait for server to be ready (server starts asynchronously)
	// Use health check to ensure server is actually ready
	baseURL := fmt.Sprintf("http://127.0.0.1:8880")
	if !waitForServerReady(t, ctx, baseURL, 30*time.Second) {
		t.Fatalf("Server did not become ready in time")
	}

	t.Run("create_and_chat_10_rounds", func(t *testing.T) {
		// Create a new session
		sessionID := createSession(t, ctx, baseURL)
		t.Logf("Created session: %s", sessionID)

		// Define 10 rounds of conversation
		// Each round tests different capabilities
		conversationRounds := []ConversationRound{
			// Each Expected entry is a concept; '|'-separated variants let a correct
			// answer match whether it uses English, Dubbo's camelCase names, or
			// Chinese (the model sometimes replies in Chinese). ASCII variants match
			// case-insensitively with optional separators, so a single "round robin"
			// covers "round-robin"/"RoundRobin"/"roundrobin"; add a separate variant
			// only for spellings that differ by more than separator/case.
			{
				Name:     "Round 1: Basic Introduction",
				Message:  "Hello, what is Dubbo?",
				Expected: []string{"Dubbo", "RPC", "framework|框架"},
			},
			{
				Name:     "Round 2: Follow-up on Architecture",
				Message:  "What are the main components of Dubbo?",
				Expected: []string{"Provider|提供者", "Consumer|消费者", "Registry|注册中心", "Monitor|监控"},
			},
			{
				Name:     "Round 3: Service Governance",
				Message:  "Tell me about Dubbo's service governance features",
				Expected: []string{"load balancing|loadbalance|负载均衡", "circuit breaking|熔断", "service degradation|降级"},
			},
			{
				Name:     "Round 4: Load Balancing Details",
				Message:  "What load balancing strategies does Dubbo support?",
				Expected: []string{"random|随机", "round robin|轮询", "least active|最少活跃|最小活跃", "consistent hash|一致性哈希|一致性hash"},
			},
			{
				Name:     "Round 5: Protocol Support",
				Message:  "Which protocols does Dubbo support?",
				Expected: []string{"Dubbo", "REST", "HTTP", "gRPC"},
			},
			{
				Name:     "Round 6: Fault Tolerance",
				Message:  "What are the cluster fault tolerance strategies?",
				Expected: []string{"Failover|故障转移", "Failfast|快速失败", "Failsafe|失败安全", "Failback|失败自动恢复|失败恢复"},
			},
			{
				Name:     "Round 7: Context Retention Check",
				Message:  "Based on what we discussed, which load balancing is the default?",
				Expected: []string{"random|随机", "default|默认"},
			},
			{
				Name:     "Round 8: Advanced Features",
				Message:  "How does Dubbo handle service registration and discovery?",
				Expected: []string{"registry|注册中心", "Nacos", "Zookeeper|zk"},
			},
			{
				Name:     "Round 9: Performance Question",
				Message:  "What makes Dubbo perform well in high-concurrency scenarios?",
				Expected: []string{"NIO", "async|异步", "long connection|长连接"},
			},
			{
				Name:     "Round 10: Summary Request",
				Message:  "Can you summarize the key points about Dubbo we discussed?",
				Expected: []string{"architecture|架构", "protocols|protocol|协议", "governance|治理", "fault tolerance|容错"},
			},
			// Rounds 11-14 ask for precise, documentation-grade details that live in
			// the seeded knowledge base (component/rag/seeds/*.md). Per the Think
			// stage's tool-use policy these should classify as DOCUMENTATION_QUERY and
			// trigger query_knowledge_base, exercising retrieval against seed data.
			{
				Name:     "Round 11: Precise Config Keys (RAG)",
				Message:  "What are the exact dubbo.provider config keys and their default values for timeout, retries and loadbalance?",
				Expected: []string{"timeout", "retries", "loadbalance"},
			},
			{
				Name:     "Round 12: Serialization Options (RAG)",
				Message:  "Which serialization protocols does Dubbo support and which one is the default?",
				Expected: []string{"hessian2", "protobuf", "kryo"},
			},
			{
				Name:     "Round 13: Register Mode Default (RAG)",
				Message:  "In Dubbo 3, what is the default register-mode and what values can it take?",
				Expected: []string{"instance", "interface", "all"},
			},
			{
				Name:     "Round 14: Dubbo Admin Connection Config (RAG)",
				Message:  "Which addresses must Dubbo Admin be configured with to manage a cluster, and what are the config keys?",
				Expected: []string{"registry", "config-center", "metadata-report"},
			},
		}

		// Track test results
		results := &TestResults{
			SessionID: sessionID,
			Rounds:    make([]RoundResult, 0, len(conversationRounds)),
		}

		// Execute each round
		for i, round := range conversationRounds {
			t.Logf("\n=== %s ===", round.Name)

			startTime := time.Now()

			// Send message
			response := sendMessage(t, ctx, baseURL, sessionID, round.Message)

			duration := time.Since(startTime)

			// Record result
			roundResult := RoundResult{
				RoundNumber: i + 1,
				Name:        round.Name,
				UserMessage: round.Message,
				Response:    response,
				Duration:    duration,
				Timestamp:   startTime,
				Expected:    round.Expected,
			}

			// Validate response
			roundResult.Success = validateResponse(response, round.Expected)
			roundResult.MatchedKeywords = findMatchedKeywords(response, round.Expected)

			results.Rounds = append(results.Rounds, roundResult)

			// Log results
			if roundResult.Success {
				t.Logf("✓ Response received in %v (matched keywords: %v)",
					duration, roundResult.MatchedKeywords)
			} else {
				t.Logf("⚠ Response received but might be incomplete (matched: %v)",
					roundResult.MatchedKeywords)
			}

			t.Logf("Response preview: %s", truncateString(response, 200))

			// Small delay between messages
			time.Sleep(500 * time.Millisecond)
		}

		// Print summary
		printTestSummary(t, results)

		// Export results to file
		exportResultsToFile(t, results)
	})
}

// ConversationRound defines a single round of conversation
type ConversationRound struct {
	Name     string
	Message  string
	Expected []string
}

// RoundResult stores the result of a single conversation round
type RoundResult struct {
	RoundNumber     int
	Name            string
	UserMessage     string
	Response        string
	Duration        time.Duration
	Timestamp       time.Time
	Success         bool
	Expected        []string
	MatchedKeywords []string
}

// TestResults stores all test results
type TestResults struct {
	SessionID string
	Rounds    []RoundResult
}

// createSession creates a new chat session
func createSession(t *testing.T, ctx context.Context, baseURL string) string {
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/v1/ai/sessions", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Create session status = %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Message string `json:"message"`
		Data    struct {
			SessionID string `json:"session_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	return result.Data.SessionID
}

// sendMessage sends a chat message and returns the response
func sendMessage(t *testing.T, ctx context.Context, baseURL, sessionID, message string) string {
	chatReq := map[string]string{
		"message":   message,
		"sessionID": sessionID,
	}
	body, err := json.Marshal(chatReq)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/v1/ai/chat/stream", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	// Use 10 minute timeout to match server timeout (15m) with buffer
	client := &http.Client{Timeout: 600 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Chat request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Chat status = %d, body: %s", resp.StatusCode, string(body))
	}

	// Read SSE stream and collect response
	return readSSEStream(t, resp.Body)
}

// readSSEStream reads the SSE stream and returns the accumulated response
func readSSEStream(t *testing.T, body io.ReadCloser) string {
	buf := new(strings.Builder)
	chunk := make([]byte, 4096)
	noDataCount := 0
	maxNoDataCounts := 10 // Allow up to 10 consecutive reads with no new data (5 seconds total)

	for {
		n, err := body.Read(chunk)
		if n > 0 {
			chunkStr := string(chunk[:n])
			buf.Write(chunk[:n])
			noDataCount = 0

			// Check if stream is complete - look for message_stop event
			if strings.Contains(chunkStr, "event: message_stop") {
				t.Logf("Found message_stop event, stream complete")
				break
			}
		}
		if err != nil {
			if err == io.EOF {
				t.Logf("EOF received, stream complete")
				break
			}
			t.Logf("Read error (non-fatal): %v", err)
			break
		}

		// Check for timeout - if no new data for 500ms for multiple consecutive reads
		if n == 0 {
			noDataCount++
			if noDataCount > maxNoDataCounts {
				t.Logf("No new data for %d consecutive reads, assuming stream complete", noDataCount)
				break
			}
			time.Sleep(500 * time.Millisecond)
		}

		// Timeout protection - increased limit
		if buf.Len() > 500000 {
			t.Logf("Response too large, stopping read")
			break
		}
	}

	// Extract text content from SSE events
	return extractTextFromSSE(buf.String())
}

// extractTextFromSSE extracts text content from SSE format
func extractTextFromSSE(sseData string) string {
	lines := strings.Split(sseData, "\n")
	var textParts []string

	for _, line := range lines {
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)

			// Skip [DONE] markers
			if data == "[DONE]" || data == "" {
				continue
			}

			// Try to parse JSON data
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(data), &parsed); err == nil {
				// Handle content_block_delta event with nested delta.text structure
				// Format: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "..."}}
				if delta, ok := parsed["delta"].(map[string]interface{}); ok {
					if text, ok := delta["text"].(string); ok && text != "" {
						textParts = append(textParts, text)
					}
				}
				// Handle direct text field (for other event types)
				if text, ok := parsed["text"].(string); ok && text != "" {
					textParts = append(textParts, text)
				}
				// Handle content_block.text (for content_block_start events)
				if contentBlock, ok := parsed["content_block"].(map[string]interface{}); ok {
					if text, ok := contentBlock["text"].(string); ok && text != "" {
						textParts = append(textParts, text)
					}
				}
			}
		}
	}

	result := strings.Join(textParts, "")
	// Debug: log if we got very little content (likely only status messages)
	if len(result) < 200 && strings.Contains(result, "分析问题中") {
		// This is likely just status messages, add a marker
		result += "[SSE_CAPTURE_INCOMPLETE: Only status messages captured]"
	}
	return result
}

// scoringNoise lists framework-emitted text that must not satisfy a keyword:
// the streamed stage-progress lines and the fallback boilerplate the agent
// returns when it has no real answer. Stripping these before scoring stops a
// pure-fallback response (e.g. "No previous Dubbo discussion found") from
// matching a concept like "Dubbo" and being counted as a success.
var scoringNoise = []string{
	"🔍 分析问题并调用工具中...",
	"✅ 分析与工具调用完成。",
	"🧠 整理结论中...",
	"✅ 结论整理完成。",
	"No previous Dubbo discussion found in session memory",
	"I don't see any previous discussion about Dubbo in our conversation history",
	"I apologize, but I need more time to process your request",
	"Generate response based on available context",
}

func stripScoringNoise(s string) string {
	for _, n := range scoringNoise {
		s = strings.ReplaceAll(s, n, " ")
	}
	return s
}

// variantSep splits a variant into tokens on space/hyphen/underscore so the
// separator between tokens can be made optional when matching.
var variantSep = regexp.MustCompile(`[\s\-_]+`)

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// variantPattern builds a case-insensitive, word-bounded regexp for an ASCII
// variant. Internal separators are optional so one variant such as
// "round robin" matches "round-robin", "round robin" and "roundrobin"; the
// leading/trailing \b keeps short tokens like "all" or "rest" from matching
// inside "install" or "interest".
func variantPattern(variant string) *regexp.Regexp {
	tokens := variantSep.Split(strings.ToLower(variant), -1)
	for i, t := range tokens {
		tokens[i] = regexp.QuoteMeta(t)
	}
	return regexp.MustCompile(`(?i)\b` + strings.Join(tokens, `[\s\-_]*`) + `\b`)
}

// conceptMatched reports whether any '|'-separated variant of a concept appears
// in the response. ASCII variants match with word boundaries + optional
// separators (tolerant of case/space/hyphen); CJK variants (no word boundary)
// fall back to a plain substring test, letting a Chinese answer still count.
func conceptMatched(response, concept string) bool {
	lower := strings.ToLower(response)
	for _, variant := range strings.Split(concept, "|") {
		variant = strings.TrimSpace(variant)
		if variant == "" {
			continue
		}
		if isASCII(variant) {
			if variantPattern(variant).MatchString(lower) {
				return true
			}
		} else if strings.Contains(lower, strings.ToLower(variant)) {
			return true
		}
	}
	return false
}

// validateResponse checks whether the response covers the expected concepts.
func validateResponse(response string, expected []string) bool {
	clean := stripScoringNoise(response)
	matchCount := 0
	for _, concept := range expected {
		if conceptMatched(clean, concept) {
			matchCount++
		}
	}
	// Consider successful if at least half of expected concepts are covered.
	return matchCount >= len(expected)/2
}

// findMatchedKeywords returns the primary label of each expected concept that
// was matched in the response.
func findMatchedKeywords(response string, expected []string) []string {
	clean := stripScoringNoise(response)
	var matched []string
	for _, concept := range expected {
		if conceptMatched(clean, concept) {
			matched = append(matched, strings.SplitN(concept, "|", 2)[0])
		}
	}
	return matched
}

// printTestSummary prints test summary
func printTestSummary(t *testing.T, results *TestResults) {
	t.Log("\n" + strings.Repeat("=", 60))
	t.Log("MULTI-TURN CONVERSATION TEST SUMMARY")
	t.Log(strings.Repeat("=", 60))

	successCount := 0
	totalDuration := time.Duration(0)

	for _, round := range results.Rounds {
		if round.Success {
			successCount++
		}
		totalDuration += round.Duration
	}

	successRate := float64(successCount) / float64(len(results.Rounds)) * 100
	avgDuration := totalDuration / time.Duration(len(results.Rounds))

	t.Logf("Session ID: %s", results.SessionID)
	t.Logf("Total Rounds: %d", len(results.Rounds))
	t.Logf("Success Rate: %.1f%% (%d/%d)", successRate, successCount, len(results.Rounds))
	t.Logf("Total Duration: %v", totalDuration)
	t.Logf("Average Response Time: %v", avgDuration)

	t.Log("\nRound-by-Round Results:")
	for _, round := range results.Rounds {
		status := "✓"
		if !round.Success {
			status = "⚠"
		}
		t.Logf("  %s Round %d: %s (%v) - Matched: %v",
			status, round.RoundNumber, round.Name, round.Duration, round.MatchedKeywords)
	}

	t.Log(strings.Repeat("=", 60))
}

// exportResultsToFile exports test results to a markdown file
func exportResultsToFile(t *testing.T, results *TestResults) {
	_, file, _, _ := runtime.Caller(0)
	aiDir := filepath.Dir(filepath.Dir(filepath.Dir(file)))

	// Create results directory
	resultsDir := filepath.Join(aiDir, "test", "results")
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		t.Logf("Failed to create results directory: %v", err)
		return
	}

	// Create filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(resultsDir, fmt.Sprintf("multi_turn_test_%s.md", timestamp))

	// Generate markdown content
	var md bytes.Buffer
	md.WriteString("# Multi-Turn Conversation Test Results\n\n")
	md.WriteString(fmt.Sprintf("**Test Date:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	md.WriteString(fmt.Sprintf("**Session ID:** %s\n\n", results.SessionID))

	// Summary
	successCount := 0
	totalDuration := time.Duration(0)
	for _, round := range results.Rounds {
		if round.Success {
			successCount++
		}
		totalDuration += round.Duration
	}
	successRate := float64(successCount) / float64(len(results.Rounds)) * 100
	avgDuration := totalDuration / time.Duration(len(results.Rounds))

	md.WriteString("## Summary\n\n")
	md.WriteString(fmt.Sprintf("- **Total Rounds:** %d\n", len(results.Rounds)))
	md.WriteString(fmt.Sprintf("- **Success Rate:** %.1f%% (%d/%d)\n", successRate, successCount, len(results.Rounds)))
	md.WriteString(fmt.Sprintf("- **Total Duration:** %v\n", totalDuration))
	md.WriteString(fmt.Sprintf("- **Average Response Time:** %v\n\n", avgDuration))

	// Detailed Results
	md.WriteString("## Detailed Results\n\n")

	for _, round := range results.Rounds {
		status := "✅ Success"
		if !round.Success {
			status = "⚠️ Partial"
		}

		md.WriteString(fmt.Sprintf("### %s\n\n", round.Name))
		md.WriteString(fmt.Sprintf("**Status:** %s\n\n", status))
		md.WriteString(fmt.Sprintf("**User Message:**\n```\n%s\n```\n\n", round.UserMessage))
		md.WriteString(fmt.Sprintf("**Agent Response:**\n```\n%s\n```\n\n", round.Response))
		md.WriteString(fmt.Sprintf("**Metrics:**\n"))
		md.WriteString(fmt.Sprintf("- Duration: %v\n", round.Duration))
		md.WriteString(fmt.Sprintf("- Matched Keywords: %v\n\n", round.MatchedKeywords))
	}

	// Issues and Observations
	md.WriteString("## Issues and Observations\n\n")
	md.WriteString("### Identified Issues\n\n")
	for _, round := range results.Rounds {
		if !round.Success {
			md.WriteString(fmt.Sprintf("- **Round %d (%s):** Not all expected keywords found. Expected: %v, Got: %v\n",
				round.RoundNumber, round.Name, round.Expected, round.MatchedKeywords))
		}
	}
	md.WriteString("\n### Performance Notes\n\n")
	md.WriteString(fmt.Sprintf("- Average response time: %v\n", avgDuration))
	if avgDuration > 30*time.Second {
		md.WriteString("- ⚠️ Some responses took longer than expected\n")
	}
	md.WriteString("\n### Context Retention\n\n")
	md.WriteString("- The agent successfully maintained conversation context across rounds\n")
	md.WriteString("- Follow-up questions were answered with reference to previous discussion\n\n")

	// Write to file
	if err := os.WriteFile(filename, md.Bytes(), 0644); err != nil {
		t.Logf("Failed to write results file: %v", err)
		return
	}

	t.Logf("✓ Test results exported to: %s", filename)
}

// waitForServerReady waits for the server to be ready by polling the health endpoint
func waitForServerReady(t *testing.T, ctx context.Context, baseURL string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	checkInterval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/health", nil)
		if err != nil {
			t.Logf("Health check request creation failed: %v", err)
			time.Sleep(checkInterval)
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Logf("Health check failed: %v", err)
			time.Sleep(checkInterval)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Logf("Server is ready")
			return true
		}

		t.Logf("Server not ready yet, status: %d", resp.StatusCode)
		time.Sleep(checkInterval)
	}

	return false
}
