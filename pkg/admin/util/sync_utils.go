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

package util

import (
	"fmt"
	"github.com/apache/dubbo-admin/pkg/admin/cache"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"strings"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common"
)

const (
	ServiceFilterKey = ".service"
	AddressFilterKey = ".address"
	IDFilterKey      = ".id"
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

func URL2ProviderList(servicesMap map[string]*common.URL) []*model.Provider {
	var providers []*model.Provider
	if servicesMap == nil {
		return providers
	}
	for id, url := range servicesMap {
		provider := URL2Provider(id, url)
		if provider != nil {
			providers = append(providers, provider)
		}
	}
	return providers
}

func FilterFromCategory(filter map[string]string) (map[string]*common.URL, error) {
	c, ok := filter[constant.CategoryKey]
	if !ok {
		return nil, fmt.Errorf("no category")
	}
	delete(filter, constant.CategoryKey)
	services, ok := cache.InterfaceRegistryCache.Load(c)
	if !ok {
		return nil, nil
	}
	servicesMap, ok := services.(*sync.Map)
	if !ok {
		return nil, fmt.Errorf("servicesMap type not *sync.Map")
	}
	return filterFromService(servicesMap, filter)
}

func filterFromService(servicesMap *sync.Map, filter map[string]string) (map[string]*common.URL, error) {
	ret := make(map[string]*common.URL)
	var err error

	s, ok := filter[ServiceFilterKey]
	if !ok {
		servicesMap.Range(func(key, value any) bool {
			service, ok := value.(map[string]*common.URL)
			if !ok {
				err = fmt.Errorf("service type not map[string]*common.URL")
				return false
			}
			filterFromURLs(service, ret, filter)
			return true
		})
	} else {
		delete(filter, ServiceFilterKey)
		value, ok := servicesMap.Load(s)
		if ok {
			service, ok := value.(map[string]*common.URL)
			if !ok {
				return nil, fmt.Errorf("service type not map[string]*common.URL")
			}
			filterFromURLs(service, ret, filter)
		}
	}
	return ret, err
}

func filterFromURLs(from, to map[string]*common.URL, filter map[string]string) {
	if from == nil || to == nil {
		return
	}
	for id, url := range from {
		match := true
		for key, value := range filter {
			if key == AddressFilterKey {
				if strings.Contains(value, constant.Colon) {
					if value != url.Location {
						match = false
						break
					}
				} else {
					if value != url.Ip {
						match = false
						break
					}
				}
			} else {
				if value != url.GetParam(key, "") {
					match = false
					break
				}
			}
		}
		if match {
			to[id] = url
		}
	}
}
