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

package services

import (
	"strings"
	"sync"

	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"github.com/apache/dubbo-admin/pkg/admin/cache/registry"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/apache/dubbo-admin/pkg/admin/cache"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	util2 "github.com/apache/dubbo-admin/pkg/admin/util"

	"dubbo.apache.org/dubbo-go/v3/common"
)

var UrlIdsMapper sync.Map

func StartSubscribe(registry registry.AdminRegistry) {
	err := registry.Subscribe(&adminNotifyListener{})
	if err != nil {
		return
	}
}

func DestroySubscribe(registry registry.AdminRegistry) {
	err := registry.Destroy()
	if err != nil {
		return
	}
}

type adminNotifyListener struct {
	mu sync.Mutex
}

func (al *adminNotifyListener) Notify(event *dubboRegistry.ServiceEvent) {
	var interfaceName string
	al.mu.Lock()
	serviceURL := event.Service
	categories := make(map[string]map[string]map[string]*common.URL)
	category := serviceURL.GetParam(constant.CategoryKey, "")
	if len(category) == 0 {
		if constant.ConsumerSide == serviceURL.GetParam(constant.Side, "") ||
			constant.ConsumerProtocol == serviceURL.Protocol {
			category = constant.ConsumersCategory
		} else {
			category = constant.ProvidersCategory
		}
	}
	if event.Action == remoting.EventTypeDel {
		if services, ok := cache.InterfaceRegistryCache.Load(category); ok {
			if services != nil {
				servicesMap, ok := services.(*sync.Map)
				if !ok {
					// servicesMap type error
					logger.Logger().Error("servicesMap type not *sync.Map")
					return
				}
				group := serviceURL.Group()
				version := serviceURL.Version()
				if constant.AnyValue != group && constant.AnyValue != version {
					deleteURL(servicesMap, serviceURL)
				} else {
					// iterator services
					servicesMap.Range(func(key, value interface{}) bool {
						if util2.GetInterface(key.(string)) == serviceURL.Service() &&
							(constant.AnyValue == group || group == util2.GetGroup(key.(string))) &&
							(constant.AnyValue == version || version == util2.GetVersion(key.(string))) {
							deleteURL(servicesMap, serviceURL)
						}
						return true
					})
				}
			}
		}
	} else {
		interfaceName = serviceURL.Service()
		var services map[string]map[string]*common.URL
		if s, ok := categories[category]; ok {
			services = s
		} else {
			services = make(map[string]map[string]*common.URL)
			categories[category] = services
		}
		service := serviceURL.ServiceKey()
		ids, found := services[service]
		if !found {
			ids = make(map[string]*common.URL)
			services[service] = ids
		}
		if md5, ok := UrlIdsMapper.Load(serviceURL.Key()); ok {
			ids[md5.(string)] = getURLToAdd(nil, serviceURL)
		} else {
			md5 := util2.Md5_16bit(serviceURL.Key())
			ids[md5] = getURLToAdd(nil, serviceURL)
			UrlIdsMapper.LoadOrStore(serviceURL.Key(), md5)
		}
	}
	// check categories size
	if len(categories) > 0 {
		for category, value := range categories {
			services, ok := cache.InterfaceRegistryCache.Load(category)
			if ok {
				servicesMap, ok := services.(*sync.Map)
				if !ok {
					// servicesMap type error
					logger.Logger().Error("servicesMap type not *sync.Map")
					return
				}
				// iterator services key set
				servicesMap.Range(func(key, inner any) bool {
					ids, ok := value[key.(string)]
					if util2.GetInterface(key.(string)) == interfaceName && !ok {
						servicesMap.Delete(key)
					} else {
						for k, v := range ids {
							inner.(map[string]*common.URL)[k] = getURLToAdd(inner.(map[string]*common.URL)[k], v)
						}
					}
					return true
				})

				for k, v := range value {
					_, ok := services.(*sync.Map).Load(k)
					if !ok {
						services.(*sync.Map).Store(k, v)
					}
				}
			} else {
				services = &sync.Map{}
				cache.InterfaceRegistryCache.Store(category, services)
				for k, v := range value {
					services.(*sync.Map).Store(k, v)
				}
			}
		}
	}

	al.mu.Unlock()
}

func getURLToAdd(url *common.URL, newURL *common.URL) *common.URL {
	if url == nil {
		newURL = newURL.Clone()
		if currentType, ok := newURL.GetNonDefaultParam(constant.RegistryType); !ok {
			newURL.SetParam(constant.RegistryType, constant.RegistryInterface)
		} else {
			newURL.SetParam(constant.RegistryType, strings.ToUpper(currentType))
		}
	} else {
		currentType, _ := url.GetNonDefaultParam(constant.RegistryType)
		changedType, _ := newURL.GetNonDefaultParam(constant.RegistryType)
		if currentType == constant.RegistryAll || currentType != changedType {
			newURL = newURL.Clone()
			newURL.SetParam(constant.RegistryType, constant.RegistryAll)
		}
	}
	return newURL
}

func deleteURL(servicesMap *sync.Map, serviceURL *common.URL) {
	if innerServices, ok := servicesMap.Load(serviceURL.Service()); ok {
		innerServicesMap := innerServices.(map[string]*common.URL)
		if len(innerServicesMap) > 0 {
			for _, url := range innerServicesMap {
				currentType, _ := url.GetNonDefaultParam(constant.RegistryType)
				changedType := serviceURL.GetParam(constant.RegistryType, constant.RegistryInterface)
				if currentType == constant.RegistryAll {
					if changedType == constant.RegistryInstance {
						url.SetParam(constant.RegistryType, constant.RegistryInterface)
					} else {
						url.SetParam(constant.RegistryType, constant.RegistryInstance)
					}
				} else if currentType == changedType {
					servicesMap.Delete(serviceURL.Service())
				}
			}
		}
	}
}
