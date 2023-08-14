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

package universal

import (
	"strings"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/registry"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
)

type ServiceMappingChangedListenerImpl struct {
	oldServiceNames *gxset.HashSet
	listener        registry.NotifyListener
	interfaceKey    string

	mux           sync.Mutex
	delSDRegistry registry.ServiceDiscovery
}

func NewMappingListener(oldServiceNames *gxset.HashSet, listener registry.NotifyListener) *ServiceMappingChangedListenerImpl {
	return &ServiceMappingChangedListenerImpl{
		listener:        listener,
		oldServiceNames: oldServiceNames,
	}
}

func parseServices(literalServices string) *gxset.HashSet {
	set := gxset.NewSet()
	if len(literalServices) == 0 {
		return set
	}
	splitServices := strings.Split(literalServices, ",")
	for _, s := range splitServices {
		if len(s) != 0 {
			set.Add(s)
		}
	}
	return set
}

//// OnEvent on ServiceMappingChangedEvent the service mapping change event
//func (lstn *ServiceMappingChangedListenerImpl) OnEvent(e observer.Event) error {
//	lstn.mux.Lock()
//
//	sm, ok := e.(*registry.ServiceMappingChangeEvent)
//	if !ok {
//		return nil
//	}
//	newServiceNames := sm.GetServiceNames()
//	oldServiceNames := lstn.oldServiceNames
//	// serviceMapping is orderly
//	if newServiceNames.Empty() || oldServiceNames.String() == newServiceNames.String() {
//		return nil
//	}
//
//	err := lstn.updateListener(lstn.interfaceKey, newServiceNames)
//	if err != nil {
//		return err
//	}
//	lstn.oldServiceNames = newServiceNames
//	lstn.mux.Unlock()
//
//	return nil
//}

//func (lstn *ServiceMappingChangedListenerImpl) updateListener(interfaceKey string, apps *gxset.HashSet) error {
//	delSDListener := NewDubboSDNotifyListener(apps)
//	delSDListener.AddListenerAndNotify(interfaceKey, lstn.listener)
//	err := lstn.delSDRegistry.AddListener(delSDListener)
//	return err
//}

// Stop on ServiceMappingChangedEvent the service mapping change event
func (lstn *ServiceMappingChangedListenerImpl) Stop() {
}
