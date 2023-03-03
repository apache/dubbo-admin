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

package util

import (
	"admin/pkg/constant"
	"admin/pkg/model"
	"dubbo.apache.org/dubbo-go/v3/common"
)

func URL2Provider(id string, url *common.URL) *model.Provider {
	if url == nil {
		return nil
	}

	return &model.Provider{
		Entity:         model.Entity{Hash: id},
		Service:        url.ServiceKey(),
		Address:        url.Location,
		Application:    url.GetParam(constant.ApplicationKey, ""),
		URL:            url.Key(),
		Parameters:     url.String(),
		Dynamic:        url.GetParamBool(constant.DynamicKey, true),
		Enabled:        url.GetParamBool(constant.EnabledKey, true),
		Serialization:  url.GetParam(constant.SerializationKey, "hessian2"),
		Timeout:        url.GetParamInt(constant.TimeoutKey, constant.DefaultTimeout),
		Weight:         url.GetParamInt(constant.WeightKey, constant.DefaultWeight),
		Username:       url.GetParam(constant.OwnerKey, ""),
		RegistrySource: model.Interface,
	}
}
