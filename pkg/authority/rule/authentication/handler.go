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

package authentication

import (
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/apache/dubbo-admin/pkg/authority/rule/connection"
	"github.com/apache/dubbo-admin/pkg/logger"
)

type Handler interface {
	Add(key string, obj *Policy)
	Get(key string) *Policy
	Update(key string, newObj *Policy)
	Delete(key string)
}

type Impl struct {
	mutex *sync.Mutex

	revision int64
	storage  *connection.Storage
	cache    map[string]*Policy
}

func NewHandler(storage *connection.Storage) *Impl {
	return &Impl{
		mutex:    &sync.Mutex{},
		storage:  storage,
		revision: 0,
		cache:    map[string]*Policy{},
	}
}

func (i *Impl) Add(key string, obj *Policy) {
	if !i.validatePolicy(obj) {
		logger.Sugar().Warnf("invalid policy, key: %s, policy: %v", key, obj)
		return
	}

	i.mutex.Lock()
	defer i.mutex.Unlock()
	if origin := i.cache[key]; reflect.DeepEqual(origin, obj) {
		return
	}

	cloned := make(map[string]*Policy, len(i.cache)+1)

	for k, v := range i.cache {
		cloned[k] = v
	}

	cloned[key] = obj

	i.cache = cloned
	atomic.AddInt64(&i.revision, 1)

	i.Notify()
}

func (i *Impl) Get(key string) *Policy {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	return i.cache[key]
}

func (i *Impl) Update(key string, newObj *Policy) {
	if !i.validatePolicy(newObj) {
		logger.Sugar().Warnf("invalid policy, key: %s, policy: %v", key, newObj)
		return
	}

	i.mutex.Lock()
	defer i.mutex.Unlock()

	if origin := i.cache[key]; reflect.DeepEqual(origin, newObj) {
		return
	}

	cloned := make(map[string]*Policy, len(i.cache))

	for k, v := range i.cache {
		cloned[k] = v
	}

	cloned[key] = newObj

	i.cache = cloned
	atomic.AddInt64(&i.revision, 1)

	i.Notify()
}

func (i *Impl) Delete(key string) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if _, ok := i.cache[key]; !ok {
		return
	}

	cloned := make(map[string]*Policy, len(i.cache)-1)

	for k, v := range i.cache {
		if k == key {
			continue
		}
		cloned[k] = v
	}

	i.cache = cloned
	atomic.AddInt64(&i.revision, 1)

	i.Notify()
}

func (i *Impl) Notify() {
	originRule := &Origin{
		revision: i.revision,
		data:     i.cache,
	}

	i.storage.LatestRules[RuleType] = originRule

	i.storage.Mutex.RLock()
	defer i.storage.Mutex.RUnlock()
	for _, c := range i.storage.Connection {
		c.RawRuleQueue.Add(originRule)
	}
}

func (i *Impl) validatePolicy(policy *Policy) bool {
	return policy != nil
}
