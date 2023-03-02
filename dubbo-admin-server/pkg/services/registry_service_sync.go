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
	"admin/pkg/cache"
	"admin/pkg/constant"
	util2 "admin/pkg/util"
	"net/url"
	"strings"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/registry"
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

func StartSubscribe(registry registry.Registry) {
	registry.Subscribe(SUBSCRIBE, adminNotifyListener{})
}

func DestroySubscribe(registry registry.Registry) {
	registry.Destroy()
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
				group := serviceUrl.Group()
				version := serviceUrl.Version()
				if constant.AnyValue != group && constant.AnyValue != version {
					services.(*sync.Map).Delete(serviceUrl.Service())
				} else {
					// iterator services
					services.(*sync.Map).Range(func(key, value interface{}) bool {
						if util2.GetInterface(key.(string)) == serviceUrl.Service() &&
							(constant.AnyValue == group || group == util2.GetGroup(key.(string))) &&
							(constant.AnyValue == version || version == util2.GetVersion(key.(string))) {
							services.(*sync.Map).Delete(key)
						}
						return true
					})
				}
			}
		}
	} else {
		interfaceName = serviceUrl.Service()
		var services map[string]map[string]*common.URL
		if s, ok := categories[category]; ok {
			services = s
		} else {
			services = make(map[string]map[string]*common.URL)
			categories[category] = services
		}
		service := serviceUrl.ServiceKey()
		ids, found := services[service]
		if !found {
			ids = make(map[string]*common.URL)
			services[service] = ids
		}
		if md5, ok := UrlIdsMapper.Load(serviceUrl.Key()); ok {
			ids[md5.(string)] = serviceUrl
		} else {
			md5 := util2.Md5_16bit(serviceUrl.Key())
			ids[md5] = serviceUrl
			UrlIdsMapper.LoadOrStore(serviceUrl.Key(), md5)
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
					if util2.GetInterface(key.(string)) == interfaceName && !ok {
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

func (adminNotifyListener) NotifyAll(events []*registry.ServiceEvent, f func()) {
	for _, event := range events {
		adminNotifyListener{}.Notify(event)
	}
}
