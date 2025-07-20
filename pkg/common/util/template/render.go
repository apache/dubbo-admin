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

package template

import (
	"strings"

	"github.com/hoisie/mustache"
)

type contextMap map[string]interface{}

func (cm contextMap) merge(other contextMap) {
	for k, v := range other {
		cm[k] = v
	}
}

func newContextMap(key, value string) contextMap {
	if !strings.Contains(key, ".") {
		return map[string]interface{}{
			key: value,
		}
	}

	parts := strings.SplitAfterN(key, ".", 2)
	return map[string]interface{}{
		parts[0][:len(parts[0])-1]: newContextMap(parts[1], value),
	}
}

func Render(template string, values map[string]string) []byte {
	ctx := contextMap{}
	for k, v := range values {
		ctx.merge(newContextMap(k, v))
	}
	data := mustache.Render(template, ctx)
	return []byte(data)
}
