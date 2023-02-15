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
	"admin/cache"
	"admin/constant"
	"admin/util"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"net/url"
	"strings"
	"sync"
)

var SUBSCRIBE *common.URL
var UrlIdsMapper sync.Map

func init() {
	queryParams := url.Values{
		constant.InterfaceKey:  {constant.AnyValue},
		constant.GroupKey:      {constant.AnyValue},
		constant.VersionKey:    {constant.AnyValue},
		constant.ClassifierKey: {constant.AnyValue},
		constant.CategoryKey: {constant.ProvidersCategory +
			"," + constant.ConsumersCategory +
			"," + constant.RoutersCategory +
			"," + constant.ConfiguratorsCategory},
		constant.EnabledKey: {constant.AnyValue},
		constant.CheckKey:   {"false"},
	}
	SUBSCRIBE, _ = common.NewURL(common.GetLocalIp()+":0",
		common.WithProtocol(constant.AdminProtocol),
		common.WithParams(queryParams),
	)
}

func startSubscribe(registry registry.Registry) {
	registry.Subscribe(SUBSCRIBE, adminNotifyListener{})
}

type adminNotifyListener struct{}

func (adminNotifyListener) Notify(event *registry.ServiceEvent) {
	//TODO implement me
	serviceUrl := event.Service
	var interfaceName string
	categories := make(map[string]map[string]map[string]*common.URL)
	category := serviceUrl.GetParam(constant.CategoryKey, "")
	if len(category) == 0 {
		if constant.ConsumerSide == serviceUrl.GetParam(constant.Side, "") ||
			constant.ConsumerProtocol == serviceUrl.Protocol {
			category = constant.ConsumersCategory
		} else {
			category = constant.ProvidersCategory
		}
	}
	if strings.EqualFold(constant.EmptyProtocol, serviceUrl.Protocol) {
		if services, ok := cache.InterfaceRegistryCache.Load(category); ok {
			if services != nil {
				group := serviceUrl.GetParam(constant.GroupKey, "")
				version := serviceUrl.GetParam(constant.VersionKey, "")
				if constant.AnyValue == group && constant.AnyValue != version {
					services.(*sync.Map).Delete(getServiceInterface(serviceUrl))
				} else {
					// iterator services
					services.(*sync.Map).Range(func(key, value interface{}) bool {
						if util.GetInterface(key.(string)) == getServiceInterface(serviceUrl) &&
							(constant.AnyValue == group || group == util.GetGroup(key.(string))) &&
							(constant.AnyValue == version || version == util.GetVersion(key.(string))) {
							services.(*sync.Map).Delete(key)
						}
						return true
					})
				}
			}
		}
	} else {
		interfaceName = getServiceInterface(serviceUrl)
		var services map[string]map[string]*common.URL
		if s, ok := categories[category]; ok {
			services = s
		} else {
			services = make(map[string]map[string]*common.URL)
			categories[category] = services
			group := serviceUrl.GetParam(constant.GroupKey, "")
			version := serviceUrl.GetParam(constant.VersionKey, "")
			service := util.BuildServiceKey(getServiceInterface(serviceUrl), group, version)
			ids, found := services[service]
			if !found {
				ids = make(map[string]*common.URL)
				services[service] = ids
			}
			if md5, ok := UrlIdsMapper.Load(serviceUrl.Key()); ok {
				ids[md5.(string)] = serviceUrl
			} else {
				md5 := util.Md5_16bit(serviceUrl.Key())
				ids[md5] = serviceUrl
				UrlIdsMapper.LoadOrStore(serviceUrl.Key(), md5)
			}
		}
	}
	// check categories size
	if len(categories) > 0 {
		for category, value := range categories {
			services, ok := cache.InterfaceRegistryCache.Load(category)
			if ok {
				// iterator services key set
				services.(*sync.Map).Range(func(key, inner any) bool {
					_, ok := value[key.(string)]
					if util.GetInterface(key.(string)) == interfaceName && ok {
						services.(*sync.Map).Delete(key)
					}
					return true
				})
			} else {
				services = &sync.Map{}
				cache.InterfaceRegistryCache.Store(category, services)
			}
			for k, v := range value {
				services.(*sync.Map).Store(k, v)
			}
		}
	}
}

func getServiceInterface(url *common.URL) string {
	path := url.Path
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	serviceInterface := url.GetParam(constant.InterfaceKey, path)
	if len(serviceInterface) == 0 || constant.AnyValue == serviceInterface {
		serviceInterface = path
	}
	return serviceInterface
}

func (adminNotifyListener) NotifyAll(events []*registry.ServiceEvent, f func()) {
	for _, event := range events {
		adminNotifyListener{}.Notify(event)
	}
}
