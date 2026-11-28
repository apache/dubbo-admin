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

package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
)

func ParseRPCInstanceMetadata(mesh, app, revision, data string) (*RPCInstanceMetadataResource, error) {
	var spec *meshproto.RPCInstanceMetadata
	if err := json.Unmarshal([]byte(data), &spec); err != nil {
		return nil, fmt.Errorf("decode metadata for %s revision %s: %w", app, revision, err)
	}
	if spec == nil {
		return nil, fmt.Errorf("metadata for %s revision %s is empty", app, revision)
	}
	if spec.App != "" && spec.App != app || spec.Revision != "" && spec.Revision != revision {
		return nil, fmt.Errorf("metadata identity does not match %s revision %s", app, revision)
	}
	spec.App, spec.Revision = app, revision
	for key, service := range spec.Services {
		if service == nil {
			return nil, fmt.Errorf("metadata service %q is empty", key)
		}
		// MatchKey appends the protocol to ServiceKey, which omits empty group/version
		// and treats version "0.0.0" as empty.
		separator := strings.LastIndexByte(key, ':')
		if separator < 0 || separator == len(key)-1 {
			return nil, fmt.Errorf("metadata service %q has no protocol", key)
		}
		name, group, version := common.ParseServiceKey(key[:separator])
		protocol := key[separator+1:]
		if name == "" || strings.ContainsAny(name, "/:") {
			return nil, fmt.Errorf("metadata service %q has no identity", key)
		}
		if service.Name == "" {
			service.Name = name
		}
		if service.Group == "" {
			service.Group = group
		}
		if service.Version == "" {
			service.Version = version
		}
		if service.Protocol == "" {
			service.Protocol = protocol
		}
		if service.Name != name || service.Group != group || service.Protocol != protocol ||
			common.ServiceKey(name, group, service.Version) != common.ServiceKey(name, group, version) {
			return nil, fmt.Errorf("metadata service %q identity does not match its key", key)
		}
	}
	res := NewRPCInstanceMetadataResourceWithAttributes(app+"@"+revision, mesh)
	res.Spec = spec
	return res, nil
}
