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

package manifest

import (
	"strconv"
	"strings"
)

// StripPrefix removes the given prefix from prefix.
func StripPrefix(path, prefix string) string {
	pl := len(strings.Split(prefix, "/"))
	pv := strings.Split(path, "/")
	return strings.Join(pv[pl:], "/")
}

func SplitSetFlag(flag string) (string, string) {
	items := strings.Split(flag, "=")
	if len(items) != 2 {
		return flag, ""
	}
	return strings.TrimSpace(items[0]), strings.TrimSpace(items[1])
}

func GetValueFromSetFlags(setFlags []string, key string) string {
	var res string
	for _, setFlag := range setFlags {
		k, v := SplitSetFlag(setFlag)
		if k == key {
			res = v
		}
	}
	return res
}

// ParseValue parses string into a value
func ParseValue(valueStr string) interface{} {
	var value interface{}
	if v, err := strconv.Atoi(valueStr); err == nil {
		value = v
	} else if v, err := strconv.ParseFloat(valueStr, 64); err == nil {
		value = v
	} else if v, err := strconv.ParseBool(valueStr); err == nil {
		value = v
	} else {
		value = strings.ReplaceAll(valueStr, "\\,", ",")
	}
	return value
}
