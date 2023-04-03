// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package render

import (
	"bufio"
	"strings"
)

// FilterFunc is used to filter some contents of manifest
type FilterFunc func(string) string

var (
	DefaultFilters = []FilterFunc{
		CommentFilter,
		SpaceFilter,
	}
)

// CommentFilter removes all comments in manifest
func CommentFilter(input string) string {
	var builder strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	return builder.String()
}

// SpaceFilter removes all leading and trailing space of manifest
func SpaceFilter(input string) string {
	return strings.TrimSpace(input)
}
