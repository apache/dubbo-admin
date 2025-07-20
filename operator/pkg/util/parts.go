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

package util

import (
	"bufio"
	"strings"
)

// SplitString splits the given yaml doc if it's multipart document.
func SplitString(yamlText string) []string {
	out := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(yamlText))
	parts := []string{}
	active := strings.Builder{}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "---") {
			parts = append(parts, active.String())
			active = strings.Builder{}
		} else {
			active.WriteString(line)
			active.WriteString("\n")
		}
	}
	if active.Len() > 0 {
		parts = append(parts, active.String())
	}
	for _, part := range parts {
		part := strings.TrimSpace(part)
		if len(part) > 0 {
			out = append(out, part)
		}
	}
	return out
}
