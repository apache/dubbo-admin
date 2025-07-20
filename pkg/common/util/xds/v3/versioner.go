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

package v3

import (
	envoy_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"google.golang.org/protobuf/proto"
)

// SnapshotVersioner assigns versions to xDS resources in a new Snapshot.
type SnapshotVersioner interface {
	Version(new, old Snapshot) Snapshot
}

// SnapshotAutoVersioner assigns versions to xDS resources in a new Snapshot
// by reusing if possible a version from the old snapshot and
// generating a new version (UUID) otherwise.
type SnapshotAutoVersioner struct {
	UUID func() string
}

func (v SnapshotAutoVersioner) Version(new, old Snapshot) Snapshot {
	if new == nil {
		return nil
	}
	for _, typ := range new.GetSupportedTypes() {
		version := new.GetVersion(typ)
		if version != "" {
			// favor a version assigned by resource generator
			continue
		}
		if old != nil && v.equal(new.GetResources(typ), old.GetResources(typ)) {
			version = old.GetVersion(typ)
		}
		if version == "" {
			version = v.UUID()
		}
		new = new.WithVersion(typ, version)
	}
	return new
}

func (_ SnapshotAutoVersioner) equal(new, old map[string]envoy_types.Resource) bool {
	if len(new) != len(old) {
		return false
	}
	for key, newValue := range new {
		if oldValue, hasOldValue := old[key]; !hasOldValue || !proto.Equal(newValue, oldValue) {
			return false
		}
	}
	return true
}
