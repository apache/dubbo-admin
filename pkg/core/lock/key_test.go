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

package lock

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildLockKeyStableAndUnambiguous(t *testing.T) {
	first := BuildLockKey("p", "a:b", "c")
	second := BuildLockKey("p", "a", "b:c")
	require.NotEqual(t, first, second)
	require.Equal(t, first, BuildLockKey("p", "a:b", "c"))
}

func TestBuildLockKeyEscapesEdgeSegments(t *testing.T) {
	cases := [][]string{
		{"p", "", ""},
		{"p", ":", "/"},
		{"p", "%", "a/b:c"},
		{"p", "服务", "规则:一/%"},
	}

	seen := map[string][]string{}
	for _, tc := range cases {
		key := BuildLockKey(tc[0], tc[1:]...)
		if existing, ok := seen[key]; ok {
			require.Equal(t, existing, tc)
		}
		seen[key] = tc
		require.Equal(t, key, BuildLockKey(tc[0], tc[1:]...))
	}
	require.Len(t, seen, len(cases))
}
