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

package counter

import (
	"sync"
)

type Counter struct {
	name string
	data map[string]int64
	mu   sync.RWMutex
}

func NewCounter(name string) *Counter {
	return &Counter{
		name: name,
		data: make(map[string]int64),
	}
}

func (c *Counter) Get() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var sum int64
	for _, v := range c.data {
		sum += v
	}
	return sum
}

func (c *Counter) GetByGroup(group string) int64 {
	if group == "" {
		group = "default"
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data[group]
}

func (c *Counter) Increment(group string) {
	if group == "" {
		group = "default"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[group]++
}

func (c *Counter) Decrement(group string) {
	if group == "" {
		group = "default"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if value, ok := c.data[group]; ok {
		value--
		if value <= 0 {
			delete(c.data, group)
		} else {
			c.data[group] = value
		}
	}
}

func (c *Counter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]int64)
}

type DistributionCounter struct {
	name string
	data map[string]map[string]int64
	mu   sync.RWMutex
}

func NewDistributionCounter(name string) *DistributionCounter {
	return &DistributionCounter{
		name: name,
		data: make(map[string]map[string]int64),
	}
}

func (c *DistributionCounter) Increment(group, key string) {
	if group == "" {
		group = "default"
	}
	if key == "" {
		key = "unknown"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data[group] == nil {
		c.data[group] = make(map[string]int64)
	}
	c.data[group][key]++
}

func (c *DistributionCounter) Decrement(group, key string) {
	if group == "" {
		group = "default"
	}
	if key == "" {
		key = "unknown"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	groupData, exists := c.data[group]
	if !exists {
		return
	}
	if value, ok := groupData[key]; ok {
		value--
		if value <= 0 {
			delete(groupData, key)
			if len(groupData) == 0 {
				delete(c.data, group)
			}
		} else {
			groupData[key] = value
		}
	}
}

func (c *DistributionCounter) GetAll() map[string]int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]int64)
	for _, groupData := range c.data {
		for k, v := range groupData {
			result[k] += v
		}
	}
	return result
}

func (c *DistributionCounter) GetByGroup(group string) map[string]int64 {
	if group == "" {
		group = "default"
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	groupData, exists := c.data[group]
	if !exists {
		return map[string]int64{}
	}
	result := make(map[string]int64, len(groupData))
	for k, v := range groupData {
		result[k] = v
	}
	return result
}

func (c *DistributionCounter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]map[string]int64)
}
