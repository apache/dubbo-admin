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

package registry

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"
)

var registries = make(map[string]func(u *common.URL) (AdminRegistry, error))

// AddRegistry sets the registry extension with @name
func AddRegistry(name string, v func(u *common.URL) (AdminRegistry, error)) {
	registries[name] = v
}

// Registry finds the registry extension with @name
func Registry(name string, config *common.URL) (AdminRegistry, error) {
	if name != "kubernetes" && name != "kube" && name != "k8s" {
		name = "universal"
	}
	if registries[name] == nil {
		panic("registry for " + name + " does not exist. please make sure that you have imported the package dubbo.apache.org/dubbo-go/v3/registry/" + name + ".")
	}
	return registries[name](config)
}

type AdminRegistry interface {
	Subscribe(listener AdminNotifyListener) error
	Destroy() error
	Delegate() dubboRegistry.Registry
}

type AdminNotifyListener interface {
	Notify(event *dubboRegistry.ServiceEvent)
}
