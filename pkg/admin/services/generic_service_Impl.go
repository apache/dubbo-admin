// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

var (
	ApplicationConfig config.ApplicationConfig
	Registry          registry.Registry
)

//func (c *ConsumerServiceImpl) FindAll() ([]*model.Consumer, error) {
//	filter := make(map[string]string)
//	filter[constant.CategoryKey] = constant.ConsumersCategory
//	servicesMap, err := util.FilterFromCategory(filter)
//	if err != nil {
//		return nil, err
//	}
//	return util.URL2ConsumerList(servicesMap), nil
//}

//func Init() {
//	registryConfig, _ := BuildRegistryConfig(Registry)
//	ApplicationConfig := &config.ApplicationConfig{}
//	ApplicationConfig.Name = "dubbo-admin"
//	ApplicationConfig.
//}

//func BuildRegistryConfig(r registry.Registry) interface{} {
//
//}

//func BuildRegistryConfig(sregistry registry.Registry) (config.RegistryConfig, error) {
//	fromUrl := registry.GetURL()
//
//	config := &config.RegistryConfig{}
//	config.Group = fromUrl.GetRawParam("group")
//
//	//address := common.URL(fromUrl.Protocol + "://" + fromUrl.PrimitiveURL)
//
//	return *config, nil
//}
