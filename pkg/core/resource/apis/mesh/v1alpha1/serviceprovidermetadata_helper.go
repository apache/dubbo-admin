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

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

func ToServiceProviderMetadataResource(mesh, name, data string) coremodel.Resource {
	metadataSpec := &meshproto.ServiceProviderMetadata{}
	err := json.Unmarshal([]byte(data), metadataSpec)
	if err != nil {
		logger.Errorf("cannot unmarshal service provider metadata %s in %s, cause: %s, raw content:\n %s,", name, mesh, err, data)
		return nil
	}
	metadataSpec.ServiceName = metadataSpec.CanonicalName
	metadataSpec.ProviderAppName = metadataSpec.Parameters[constants.Application]
	metadataSpec.Version = metadataSpec.Parameters[constants.VersionKey]
	metadataSpec.Group = metadataSpec.Parameters[constants.GroupKey]
	serviceKey := BuildServiceKey(metadataSpec.ServiceName, metadataSpec.Version, metadataSpec.Group, metadataSpec.ProviderAppName)
	metadataRes := NewServiceProviderMetadataResourceWithAttributes(serviceKey, mesh)
	metadataRes.Spec = metadataSpec
	return metadataRes
}

func ToServiceProviderMetadataRes(mesh, data string) coremodel.Resource {
	return ToServiceProviderMetadataResource(mesh, "", data)
}
