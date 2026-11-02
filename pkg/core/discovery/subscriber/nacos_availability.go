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

package subscriber

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/core/events"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func nacosRPCInstance(service *meshresource.NacosServiceResource, instance *meshproto.NacosInstance) *meshresource.RPCInstanceResource {
	res := meshresource.ToRPCInstance(service.Mesh, service.Name, instance.Ip, instance.Port, instance.Metadata)
	available := instance.Healthy && instance.Enabled && instance.Weight > 0
	res.Spec.Available = &available
	return res
}

func (n *NacosServiceEventSubscriber) reconcileNacosMetadata(service *meshresource.NacosServiceResource) error {
	next := make(map[string]*meshresource.RPCInstanceMetadataResource)
	prefix := service.Name + "@"
	if strings.HasPrefix(service.Name, "providers:") {
		prefix = "interface:" + service.Name + ":"
		exported, err := nacosInterfaceExports(service)
		if err != nil {
			return err
		}
		next = exported
	} else if service.Spec != nil {
		for _, spec := range service.Spec.Revisions {
			if spec == nil || spec.App != service.Name || spec.Revision == "" {
				continue
			}
			resource := meshresource.NewRPCInstanceMetadataResourceWithAttributes(service.Name+"@"+spec.Revision, service.Mesh)
			resource.Spec = spec
			next[resource.ResourceKey()] = resource
		}
	}
	return n.syncNacosMetadata(service.Mesh, prefix, next)
}

func (n *NacosServiceEventSubscriber) deleteNacosMetadata(service *meshresource.NacosServiceResource) error {
	prefix := service.Name + "@"
	if strings.HasPrefix(service.Name, "providers:") {
		prefix = "interface:" + service.Name + ":"
	}
	return n.syncNacosMetadata(service.Mesh, prefix, nil)
}

func (n *NacosServiceEventSubscriber) syncNacosMetadata(mesh, namePrefix string, next map[string]*meshresource.RPCInstanceMetadataResource) error {
	st, err := n.storeRouter.ResourceKindRoute(meshresource.RPCInstanceMetadataKind)
	if err != nil {
		return err
	}
	resources, err := st.ListByIndexes([]index.IndexCondition{{IndexName: index.ByMeshIndex, Value: mesh, Operator: index.Equals}})
	if err != nil {
		return err
	}
	for _, item := range resources {
		old, ok := item.(*meshresource.RPCInstanceMetadataResource)
		if !ok {
			return fmt.Errorf("unexpected metadata resource %T", item)
		}
		if !strings.HasPrefix(old.Name, namePrefix) {
			continue
		}
		newItem, exists := next[old.ResourceKey()]
		if !exists {
			if err := st.Delete(old); err != nil {
				return err
			}
			n.emitter.Send(events.NewResourceChangedEvent(cache.Deleted, old, nil))
			continue
		}
		if !reflect.DeepEqual(old.Spec, newItem.Spec) {
			if err := st.Update(newItem); err != nil {
				return err
			}
			n.emitter.Send(events.NewResourceChangedEvent(cache.Updated, old, newItem))
		}
		delete(next, old.ResourceKey())
	}
	for _, item := range next {
		if err := st.Add(item); err != nil {
			return err
		}
		n.emitter.Send(events.NewResourceChangedEvent(cache.Added, nil, item))
	}
	return nil
}

func nacosInterfaceExports(service *meshresource.NacosServiceResource) (map[string]*meshresource.RPCInstanceMetadataResource, error) {
	next := make(map[string]*meshresource.RPCInstanceMetadataResource)
	if service.Spec == nil {
		return next, nil
	}
	parts := strings.SplitN(strings.TrimPrefix(service.Name, "providers:"), ":", 3)
	if len(parts) == 0 || parts[0] == "" {
		return nil, fmt.Errorf("invalid provider registration %q", service.Name)
	}
	version, group := "", ""
	if len(parts) > 1 {
		version = parts[1]
	}
	if len(parts) > 2 {
		group = parts[2]
	}
	for _, instance := range service.Spec.Instances {
		if instance == nil {
			continue
		}
		app := instance.Metadata[constants.Application]
		if app == "" {
			var params map[string]string
			if err := json.Unmarshal([]byte(instance.Metadata[constants.URLParamsKey]), &params); err == nil {
				app = params[constants.Application]
			}
		}
		rpc := nacosRPCInstance(service, instance)
		name := fmt.Sprintf("interface:%s:%s:%s:%d", service.Name, app, instance.Ip, instance.Port)
		resource := meshresource.NewRPCInstanceMetadataResourceWithAttributes(name, service.Mesh)
		protocol := rpc.Spec.Protocol
		if protocol == "" {
			protocol = instance.Metadata[constants.ProtocolKey]
		}
		resource.Spec = &meshproto.RPCInstanceMetadata{
			App: app, InstanceName: rpc.Name, Available: rpc.Spec.Available,
			Services: map[string]*meshproto.ServiceInfo{
				meshresource.BuildServiceIdentityKey(parts[0], version, group): {
					Name: parts[0], Version: version, Group: group, Protocol: protocol, Port: instance.Port,
				},
			},
		}
		next[resource.ResourceKey()] = resource
	}
	return next, nil
}
