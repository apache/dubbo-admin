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

package crdclient

import (
	"reflect"

	"github.com/apache/dubbo-admin/pkg/core/schema/collection"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type EventHandler struct {
	Resource Handler
}

// cacheHandler abstracts the logic of an informer with a set of handlers. Handlers can be added at runtime
// and will be invoked on each informer event.
type cacheHandler struct {
	client   *Client
	informer cache.SharedIndexInformer
	schema   collection.Schema
	handlers []EventHandler
	lister   func(namespace string) cache.GenericNamespaceLister
}

func (h *cacheHandler) onEvent(curr interface{}) error {
	if err := h.client.checkReadyForEvents(curr); err != nil {
		return err
	}

	for _, f := range h.handlers {
		err := f.Resource.NotifyWithIndex(h.schema)
		if err != nil {
			return err
		}
	}
	return nil
}

func createCacheHandler(cl *Client, schema collection.Schema, i informers.GenericInformer) *cacheHandler {
	h := &cacheHandler{
		client:   cl,
		informer: i.Informer(),
		schema:   schema,
	}
	h.lister = func(namespace string) cache.GenericNamespaceLister {
		if schema.Resource().IsClusterScoped() {
			return i.Lister()
		}
		return i.Lister().ByNamespace(namespace)
	}
	_, err := i.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cl.queue.Push(func() error {
				return h.onEvent(obj)
			})
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if reflect.DeepEqual(oldObj, newObj) {
				return
			}
			cl.queue.Push(func() error {
				return h.onEvent(newObj)
			})
		},
		DeleteFunc: func(obj interface{}) {
			cl.queue.Push(func() error {
				return h.onEvent(obj)
			})
		},
	})
	if err != nil {
		return nil
	}
	return h
}
