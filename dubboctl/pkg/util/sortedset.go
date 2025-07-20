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
	"sort"
	"sync"
)

// sorted set of strings.
//
// write-optimized and suitable only for fairly small values of N.
// Should this increase dramatically in size, a different implementation,
// such as a linked list, might be more appropriate.
type sortedSet struct {
	members map[string]bool
	sync.Mutex
}

func NewSortedSet() *sortedSet {
	return &sortedSet{
		members: make(map[string]bool),
	}
}

func (s *sortedSet) Add(value string) {
	s.Lock()
	s.members[value] = true
	s.Unlock()
}

func (s *sortedSet) Remove(value string) {
	s.Lock()
	delete(s.members, value)
	s.Unlock()
}

func (s *sortedSet) Items() []string {
	s.Lock()
	defer s.Unlock()
	n := []string{}
	for k := range s.members {
		n = append(n, k)
	}
	sort.Strings(n)
	return n
}
