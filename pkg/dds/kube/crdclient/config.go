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
	"github.com/apache/dubbo-admin/pkg/core/model"
	"github.com/apache/dubbo-admin/pkg/core/schema/collection"
)

type ConfigStore interface {
	// Schemas exposes the configuration type schema known by the config store.
	// The type schema defines the bidrectional mapping between configuration
	// types and the protobuf encoding schema.
	Schemas() collection.Schemas

	// Get retrieves a configuration element by a type and a key
	Get(typ model.GroupVersionKind, name, namespace string) *model.Config

	// List returns objects by type and namespace.
	// Use "" for the namespace to list across namespaces.
	List(typ model.GroupVersionKind, namespace string) ([]model.Config, error)
}

// ConfigStoreCache TODO Maybe we can reuse the cache in client-go?
type ConfigStoreCache interface {
	ConfigStore

	// RegisterEventHandler adds a handler to receive config update events for a
	// configuration type
	RegisterEventHandler(kind model.GroupVersionKind, handler EventHandler)

	// Start until a signal is received
	Start(stop <-chan struct{}) error

	// HasSynced returns true after initial cache synchronization is complete
	HasSynced() bool
}

type Event int

const (
	// EventAdd is sent when an object is added
	EventAdd Event = iota

	// EventUpdate is sent when an object is modified
	// Captures the modified object
	EventUpdate

	// EventDelete is sent when an object is deleted
	// Captures the object at the last known state
	EventDelete
)

const (
	// NamespaceAll is a designated symbol for listing across all namespaces
	NamespaceAll = ""
)
