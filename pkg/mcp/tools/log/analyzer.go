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

package log

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var volatileTokenPattern = regexp.MustCompile(`(?i)\b[0-9a-f]{8,}\b|\b\d+\b`)

func analyzeErrors(logs []LogItem, sourceEngine string) *AnalyzeErrorLogsResp {
	patternsByName := map[string]*ErrorPattern{}
	for _, item := range logs {
		if !isErrorLog(item) {
			continue
		}
		patternName := normalizeMessagePattern(item.Message)
		pattern := patternsByName[patternName]
		if pattern == nil {
			pattern = &ErrorPattern{
				Pattern:   patternName,
				Example:   item.Message,
				FirstSeen: item.Timestamp,
				LastSeen:  item.Timestamp,
			}
			patternsByName[patternName] = pattern
		}
		pattern.Count++
		if pattern.FirstSeen == "" || item.Timestamp < pattern.FirstSeen {
			pattern.FirstSeen = item.Timestamp
		}
		if pattern.LastSeen == "" || item.Timestamp > pattern.LastSeen {
			pattern.LastSeen = item.Timestamp
		}
		if len(pattern.Examples) < 3 {
			pattern.Examples = append(pattern.Examples, item)
		}
	}

	patterns := make([]ErrorPattern, 0, len(patternsByName))
	total := 0
	for _, pattern := range patternsByName {
		total += pattern.Count
		patterns = append(patterns, *pattern)
	}
	sort.Slice(patterns, func(i, j int) bool {
		if patterns[i].Count == patterns[j].Count {
			return patterns[i].Pattern < patterns[j].Pattern
		}
		return patterns[i].Count > patterns[j].Count
	})

	return &AnalyzeErrorLogsResp{
		Summary:      fmt.Sprintf("found %d error log entries across %d patterns", total, len(patterns)),
		TotalErrors:  total,
		Patterns:     patterns,
		SourceEngine: sourceEngine,
	}
}

func isErrorLog(item LogItem) bool {
	if strings.EqualFold(item.Severity, "error") {
		return true
	}
	message := strings.ToLower(item.Message)
	return strings.Contains(message, "error") || strings.Contains(message, "exception") || strings.Contains(message, "failed")
}

func normalizeMessagePattern(message string) string {
	pattern := volatileTokenPattern.ReplaceAllString(message, "?")
	pattern = strings.Join(strings.Fields(pattern), " ")
	if pattern == "" {
		return "unknown"
	}
	return pattern
}
