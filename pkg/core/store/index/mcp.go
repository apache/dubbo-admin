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

package index

import (
	"reflect"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

const (
	ByMCPServerName         = "idx_mcp_server_name"
	ByMCPCredentialID       = "idx_mcp_credential_id"
	ByMCPCredentialServerID = "idx_mcp_credential_server_id"
)

func init() {
	RegisterIndexers(meshresource.MCPServerKind, map[string]cache.IndexFunc{
		ByMCPServerName: byMCPServerName,
	})
	RegisterIndexers(meshresource.MCPCredentialKind, map[string]cache.IndexFunc{
		ByMCPCredentialID:       byMCPCredentialID,
		ByMCPCredentialServerID: byMCPCredentialServerID,
	})
}

func byMCPServerName(obj interface{}) ([]string, error) {
	resource, ok := obj.(*meshresource.MCPServerResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.MCPServerKind, reflect.TypeOf(obj).Name())
	}
	return []string{resource.Name}, nil
}

func byMCPCredentialID(obj interface{}) ([]string, error) {
	resource, ok := obj.(*meshresource.MCPCredentialResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.MCPCredentialKind, reflect.TypeOf(obj).Name())
	}
	return []string{resource.Name}, nil
}

func byMCPCredentialServerID(obj interface{}) ([]string, error) {
	resource, ok := obj.(*meshresource.MCPCredentialResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.MCPCredentialKind, reflect.TypeOf(obj).Name())
	}
	if resource.Spec == nil || resource.Spec.ServerId == "" {
		return []string{}, nil
	}
	return []string{resource.Spec.ServerId}, nil
}
