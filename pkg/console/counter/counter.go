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
	"sync/atomic"
)

type Counter struct {
	name  string
	value atomic.Int64
}

func NewCounter(name string) *Counter {
	return &Counter{name: name}
}

func (c *Counter) Get() int64 {
	return c.value.Load()
}

func (c *Counter) Increment() {
	c.value.Add(1)
}

func (c *Counter) Decrement() {
	for {
		current := c.value.Load()
		if current == 0 {
			return
		}
		if c.value.CompareAndSwap(current, current-1) {
			return
		}
	}
}

func (c *Counter) Reset() {
	c.value.Store(0)
}

type DistributionCounter struct {
	name string
	data map[string]int64
	mu   sync.RWMutex
}

func NewDistributionCounter(name string) *DistributionCounter {
	return &DistributionCounter{
		name: name,
		data: make(map[string]int64),
	}
}

func (c *DistributionCounter) Increment(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key]++
}

func (c *DistributionCounter) Decrement(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if value, ok := c.data[key]; ok {
		value--
		if value <= 0 {
			delete(c.data, key)
		} else {
			c.data[key] = value
		}
	}
}

func (c *DistributionCounter) GetAll() map[string]int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]int64, len(c.data))
	for k, v := range c.data {
		result[k] = v
	}
	return result
}

func (c *DistributionCounter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]int64)
}
