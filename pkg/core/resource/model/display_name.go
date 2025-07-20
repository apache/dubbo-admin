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

package model

import (
	"strings"
	"unicode"
)

func DisplayName(resType string) string {
	displayName := ""
	for i, c := range resType {
		if unicode.IsUpper(c) && i != 0 {
			displayName += " "
		}
		displayName += string(c)
	}
	return displayName
}

func PluralType(resType string) string {
	switch {
	case strings.HasSuffix(resType, "ay"):
		return resType + "s"
	case strings.HasSuffix(resType, "y"):
		return strings.TrimSuffix(resType, "y") + "ies"
	case strings.HasSuffix(resType, "s"), strings.HasSuffix(resType, "sh"), strings.HasSuffix(resType, "ch"):
		return resType + "es"
	default:
		return resType + "s"
	}
}
