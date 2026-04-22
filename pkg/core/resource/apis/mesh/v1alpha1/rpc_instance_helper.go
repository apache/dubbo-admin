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
	"strconv"
	"time"

	set "github.com/duke-git/lancet/v2/datastructure/set"
	"github.com/duke-git/lancet/v2/maputil"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/core/logger"
)

func ToRPCInstance(
	mesh string, appName string, ip string,
	port int64, metadata map[string]string) *RPCInstanceResource {
	resName := BuildInstanceResName(appName, ip, port)
	var registerTime string
	timestamp, err := strconv.ParseInt(metadata[constants.TimestampKey], 10, 64)
	if err == nil {
		registerTime = time.UnixMilli(timestamp).Format(constants.TimeFormatStr)
	}
	revision := metadata[constants.MetadataRevisionKey]
	metadataStorageType := metadata[constants.MetadataStorageTypeKey]
	urlParams, exists := metadata[constants.URLParamsKey]
	releaseVersion := ""
	protocol := ""
	serialization := ""
	preferSerialization := ""
	if exists {
		paramsMap := make(map[string]string)
		err := json.Unmarshal([]byte(urlParams), &paramsMap)
		if err != nil {
			logger.Warnf("parse url params failed, raw url params string: %s, cause: %v", urlParams, err)
		}
		releaseVersion = paramsMap[constants.ReleaseKey]
		protocol = paramsMap[constants.ProtocolKey]
		serialization = paramsMap[constants.SerializationKey]
		preferSerialization = paramsMap[constants.PreferSerializationKey]
	}
	var endpoints []*meshproto.Endpoint
	err = json.Unmarshal([]byte(metadata[constants.EndpointsKey]), &endpoints)
	if err != nil {
		logger.Warnf("parse endpoints failed, raw endpoints string: %s, cause: %v", metadata[constants.EndpointsKey], err)
	}
	res := NewRPCInstanceResourceWithAttributes(resName, mesh)
	res.Spec = &meshproto.RPCInstance{
		Name:                resName,
		AppName:             appName,
		Ip:                  ip,
		Port:                port,
		RegisterTime:        registerTime,
		UnregisterTime:      "",
		Revision:            revision,
		MetadataStorageType: metadataStorageType,
		ReleaseVersion:      releaseVersion,
		Protocol:            protocol,
		Serialization:       serialization,
		PreferSerialization: preferSerialization,
		Tags:                getRPCInstanceTags(metadata),
		Endpoints:           endpoints,
		Metadata:            metadata,
	}
	return res
}

func getRPCInstanceTags(metadata map[string]string) map[string]string {
	knownKeys := set.New[string](constants.URLParamsKey, constants.EndpointsKey,
		constants.MetadataRevisionKey, constants.MetadataStorageTypeKey, constants.TimestampKey)
	return maputil.Filter(metadata, func(key string, value string) bool {
		return !knownKeys.Contain(key)
	})
}
