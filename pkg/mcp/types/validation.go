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

package types

import "fmt"

// ValidateRequired 验证必需参数是否存在且非空
func ValidateRequired(schema InputSchema, args map[string]any) error {
	if args == nil {
		args = make(map[string]any)
	}

	for _, required := range schema.Required {
		val, exists := args[required]
		if !exists {
			return fmt.Errorf("missing required parameter: %s", required)
		}

		if IsEmpty(val) {
			return fmt.Errorf("required parameter %s cannot be empty", required)
		}
	}
	return nil
}

// IsEmpty 判断值是否为空
func IsEmpty(val any) bool {
	switch v := val.(type) {
	case string:
		return v == ""
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	case nil:
		return true
	default:
		return false
	}
}
