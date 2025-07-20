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

package yaml

import (
	"regexp"
	"strings"
)

var sep = regexp.MustCompile("(?:^|\\s*\n)---\\s*")

// SplitYAML takes YAMLs separated by `---` line and splits it into multiple YAMLs. Empty entries are ignored
func SplitYAML(yamls string) []string {
	var result []string
	// Making sure that any extra whitespace in YAML stream doesn't interfere in splitting documents correctly.
	trimYAMLs := strings.TrimSpace(yamls)
	docs := sep.Split(trimYAMLs, -1)
	for _, doc := range docs {
		if doc == "" {
			continue
		}
		doc = strings.TrimSpace(doc)
		result = append(result, doc)
	}
	return result
}
