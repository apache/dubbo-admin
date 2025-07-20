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

package ptr

// Of returns a pointer to the input. In most cases, callers should just do &t. However, in some cases
// Go cannot take a pointer. For example, `pointer.Of(f())`.
func Of[T any](t T) *T {
	return &t
}

// Empty returns an empty T type
func Empty[T any]() T {
	var empty T
	return empty
}

// NonEmptyOrDefault returns t if its non-empty, or else def.
func NonEmptyOrDefault[T comparable](t T, def T) T {
	var empty T
	if t != empty {
		return t
	}
	return def
}
