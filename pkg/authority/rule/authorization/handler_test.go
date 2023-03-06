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

package authorization_test

import (
	"strconv"
	"sync"
	"testing"

	"github.com/apache/dubbo-admin/pkg/authority/rule/authorization"
	"github.com/apache/dubbo-admin/pkg/authority/rule/connection"
	"k8s.io/client-go/util/workqueue"
)

func TestAdd(t *testing.T) {
	t.Parallel()

	storage := connection.NewStorage()
	wq := workqueue.NewNamed("raw-rule")
	storage.Connection = []*connection.Connection{
		{
			RawRuleQueue: wq,
		},
	}

	handler := authorization.NewHandler(storage)

	handler.Add("test", nil)

	if handler.Get("test") != nil {
		t.Error("expected policy to be nil")
	}

	if wq.Len() != 0 {
		t.Error("expected queue length to be 0")
	}

	policy := &authorization.Policy{}
	handler.Add("test", policy)

	if handler.Get("test") != policy {
		t.Error("expected policy to be added")
	}

	if handler.Get("test2") != nil {
		t.Error("expected policy to be nil")
	}

	if wq.Len() != 1 {
		t.Error("expected queue length to be 1")
	}

	handler.Add("test2", policy)

	if handler.Get("test2") != policy {
		t.Error("expected policy to be added")
	}

	if wq.Len() != 2 {
		t.Error("expected queue length to be 2")
	}

	handler.Add("test", policy)

	if handler.Get("test") != policy {
		t.Error("expected policy to be added")
	}

	if wq.Len() != 2 {
		t.Error("expected queue length to be 2")
	}

	policies := make(map[string]*authorization.Policy, 1000)

	for i := 0; i < 1000; i++ {
		policies["test"+strconv.Itoa(i)] = &authorization.Policy{
			Name: "test" + strconv.Itoa(i),
		}
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(policies))
	for k, v := range policies {
		k := k
		v := v
		go func() {
			handler.Add(k, v)
			wg.Done()
		}()
	}
	wg.Wait()

	for k, v := range policies {
		if handler.Get(k) != v {
			t.Error("expected policy to be added, key: " + k)
		}
	}

	if wq.Len() != 1002 {
		t.Error("expected queue length to be 1002, but got " + strconv.Itoa(wq.Len()))
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	storage := connection.NewStorage()
	wq := workqueue.NewNamed("raw-rule")
	storage.Connection = []*connection.Connection{
		{
			RawRuleQueue: wq,
		},
	}

	handler := authorization.NewHandler(storage)

	handler.Update("test", nil)

	if handler.Get("test") != nil {
		t.Error("expected policy to be nil")
	}

	if wq.Len() != 0 {
		t.Error("expected queue length to be 0")
	}

	policy := &authorization.Policy{}
	handler.Update("test", policy)

	if handler.Get("test") != policy {
		t.Error("expected policy to be added")
	}

	if handler.Get("test2") != nil {
		t.Error("expected policy to be nil")
	}

	if wq.Len() != 1 {
		t.Error("expected queue length to be 1")
	}

	handler.Update("test2", policy)

	if handler.Get("test2") != policy {
		t.Error("expected policy to be added")
	}

	if wq.Len() != 2 {
		t.Error("expected queue length to be 2")
	}

	handler.Update("test", policy)

	if handler.Get("test") != policy {
		t.Error("expected policy to be added")
	}

	if wq.Len() != 2 {
		t.Error("expected queue length to be 2")
	}

	policies := make(map[string]*authorization.Policy, 1000)

	for i := 0; i < 1000; i++ {
		policies["test"+strconv.Itoa(i)] = &authorization.Policy{
			Name: "test" + strconv.Itoa(i),
		}
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(policies))
	for k, v := range policies {
		k := k
		v := v
		go func() {
			handler.Update(k, v)
			wg.Done()
		}()
	}
	wg.Wait()

	for k, v := range policies {
		if handler.Get(k) != v {
			t.Error("expected policy to be added, key: " + k)
		}
	}

	if wq.Len() != 1002 {
		t.Error("expected queue length to be 10002, but got " + strconv.Itoa(wq.Len()))
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	storage := connection.NewStorage()
	wq := workqueue.NewNamed("raw-rule")
	storage.Connection = []*connection.Connection{
		{
			RawRuleQueue: wq,
		},
	}

	handler := authorization.NewHandler(storage)

	policy := &authorization.Policy{}
	handler.Add("test", policy)

	if handler.Get("test") != policy {
		t.Error("expected policy to be added")
	}

	if handler.Get("test2") != nil {
		t.Error("expected policy to be nil")
	}

	if wq.Len() != 1 {
		t.Error("expected queue length to be 1")
	}

	handler.Add("test2", policy)

	if handler.Get("test2") != policy {
		t.Error("expected policy to be added")
	}

	if wq.Len() != 2 {
		t.Error("expected queue length to be 2")
	}

	handler.Add("test", policy)

	if handler.Get("test") != policy {
		t.Error("expected policy to be added")
	}

	if wq.Len() != 2 {
		t.Error("expected queue length to be 2")
	}

	handler.Delete("test")

	if handler.Get("test") != nil {
		t.Error("expected policy to be nil")
	}

	if wq.Len() != 3 {
		t.Error("expected queue length to be 3")
	}

	handler.Delete("test2")

	if handler.Get("test2") != nil {
		t.Error("expected policy to be nil")
	}

	if wq.Len() != 4 {
		t.Error("expected queue length to be 4")
	}

	handler.Delete("test")

	if handler.Get("test") != nil {
		t.Error("expected policy to be nil")
	}

	if wq.Len() != 4 {
		t.Error("expected queue length to be 4")
	}

	for i := 0; i < 1000; i++ {
		handler.Add("test"+strconv.Itoa(i), &authorization.Policy{
			Name: "test" + strconv.Itoa(i),
		})
	}

	wg := &sync.WaitGroup{}
	wg.Add(10000)
	for i := 0; i < 10; i++ {
		go func() {
			for i := 0; i < 1000; i++ {
				i := i
				go func() {
					handler.Delete("test" + strconv.Itoa(i))
					wg.Done()
				}()
			}
		}()
	}
	wg.Wait()

	for i := 0; i < 1000; i++ {
		if handler.Get("test"+strconv.Itoa(i)) != nil {
			t.Error("expected policy to be deleted, key: " + "test" + strconv.Itoa(i))
		}
	}

	if wq.Len() != 2004 {
		t.Errorf("expected queue length to be 1004, got %d", wq.Len())
	}
}
