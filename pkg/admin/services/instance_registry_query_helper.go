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
	"dubbo.apache.org/dubbo-go/v3/common"
	"fmt"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"sync"
)

type InstanceRegistryQueryHelper struct {
}

func (p *InstanceRegistryQueryHelper) FindVersionInApplication(application string) (string, error) {
	var (
		version string
	)
	var instanceRegistryCache sync.Map
	appInterfaceMap, ok := instanceRegistryCache.Load(constant.ProvidersCategory)
	if !ok || appInterfaceMap == nil {
		return "", fmt.Errorf("can't find application")
	}
	servicesMap, ok := appInterfaceMap.(*sync.Map)
	if !ok {
		return "", fmt.Errorf("servicesMap type not *sync.Map")
	}
	_, ok = servicesMap.Load(application)
	if !ok {
		return "", fmt.Errorf("can't find application")
	}

	servicesMap.Range(func(key, value any) bool {
		serviceName, ok := key.(string)
		if !ok {
			_ = fmt.Errorf("service name not string")
			return false
		}
		if serviceName == application {
			// java 中是 InstanceAddressURL，没找到对应实现，暂时用其父类 URL
			urlsMap, ok := value.(map[string][]common.URL)
			if !ok {
				_ = fmt.Errorf("service type not map[string]*common.URL")
				return false
			}
			for _, urls := range urlsMap {
				if len(urls) != 0 {
					version = urls[0].GetParam(constant.SPECIFICATION_VERSION_KEY, "3.0.0")
					break
				}
			}
		}
		return true
	})
	return version, nil
}
