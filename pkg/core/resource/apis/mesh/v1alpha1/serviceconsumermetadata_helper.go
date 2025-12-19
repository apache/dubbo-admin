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

func ToServiceConsumerMetadataByRawData(mesh string, data string) coremodel.Resource {
	metadataMap := make(map[string]string)
	err := json.Unmarshal([]byte(data), &metadataMap)
	if err != nil {
		logger.Errorf("parse service consumer metadata failed, cause: %v, raw data:\n %s", err, data)
		return nil
	}
	return ToServiceConsumerMetadataByMap(metadataMap, mesh)
}

func ToServiceConsumerMetadataByMap(metadata map[string]string, mesh string) *ServiceConsumerMetadataResource {
	consumerAppName, exists := metadata[constants.Application]
	if !exists {
		logger.Errorf("service consumer metadata is invalid because no application name found, raw metadata: %v", metadata)
	}
	serviceName, exists := metadata[constants.InterfaceKey]
	if !exists {
		logger.Errorf("service consumer metadata is invalid because no service name found, raw metadata: %v", metadata)
		return nil
	}
	version := metadata[constants.VersionKey]
	group := metadata[constants.GroupKey]
	resKey := BuildServiceKey(serviceName, version, group, consumerAppName)
	res := NewServiceConsumerMetadataResourceWithAttributes(resKey, mesh)
	res.Spec = &meshproto.ServiceConsumerMetadata{
		ServiceName:     serviceName,
		ConsumerAppName: consumerAppName,
		Version:         version,
		Group:           group,
		Metadata:        metadata,
	}
	return res
}
