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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/convertor"
	set "github.com/duke-git/lancet/v2/datastructure/set"
	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

type NacosServiceEventSubscriber struct {
	emitter     events.Emitter
	storeRouter store.Router
}

func NewNacosServiceEventSubscriber(eventEmitter events.Emitter, storeRouter store.Router) *NacosServiceEventSubscriber {
	return &NacosServiceEventSubscriber{
		emitter:     eventEmitter,
		storeRouter: storeRouter,
	}
}

func (n *NacosServiceEventSubscriber) ResourceKind() coremodel.ResourceKind {
	return meshresource.NacosServiceKind
}

func (n *NacosServiceEventSubscriber) Name() string {
	return "Nacos2Discovery-" + n.ResourceKind().ToString()
}

func (n *NacosServiceEventSubscriber) ProcessEvent(event events.Event) error {
	newObj, ok := event.NewObj().(*meshresource.NacosServiceResource)
	if !ok && event.NewObj() != nil {
		return bizerror.NewAssertionError(reflect.TypeOf(newObj), event.NewObj())
	}
	oldObj, ok := event.OldObj().(*meshresource.NacosServiceResource)
	if !ok && event.OldObj() != nil {
		return bizerror.NewAssertionError(reflect.TypeOf(oldObj), event.OldObj())
	}
	var processErr error
	switch event.Type() {
	case cache.Added, cache.Updated, cache.Replaced, cache.Sync:
		if newObj == nil {
			errStr := "process nacos service upsert event, but new obj is nil, skipped processing"
			logger.Errorf(errStr)
			return bizerror.New(bizerror.EventError, errStr)
		}
		processErr = n.processUpsert(newObj)
	case cache.Deleted:
		if oldObj == nil {
			errStr := "process nacos service delete event, but old obj is nil, skipped processing"
			logger.Errorf(errStr)
			return bizerror.New(bizerror.EventError, errStr)
		}
		processErr = n.processDelete(oldObj)
	}
	if processErr != nil {
		logger.Errorf("process nacos service event failed, cause: %s, event: %s", processErr.Error(), event.String())
		return processErr
	}
	logger.Infof("process nacos service event successfully, event: %s", event.String())
	return nil
}

func (n *NacosServiceEventSubscriber) processUpsert(serviceRes *meshresource.NacosServiceResource) error {
	providerRe := regexp.MustCompile(`^providers:[\w.]+(?::[\w.]*:|::[\w.]*)?$`)
	consumerRe := regexp.MustCompile(`^consumers:[\w.]+(?::[\w.]*:|::[\w.]*)?$`)
	if providerRe.MatchString(serviceRes.Name) {
		logger.Infof("interface-level service ignored, service name: %s", serviceRes.Name)
		return nil
	}
	if consumerRe.MatchString(serviceRes.Name) {
		return n.processConsumerMetadataUpsert(serviceRes)
	}
	return n.processRPCInstanceUpsert(serviceRes)

}

func (n *NacosServiceEventSubscriber) processConsumerMetadataUpsert(serviceRes *meshresource.NacosServiceResource) error {
	serviceName, err := parseServiceName(serviceRes.Name)
	if err != nil {
		return bizerror.Wrap(err, bizerror.UnknownError, "parse service name error, raw nacos service is"+serviceRes.String())
	}
	convertFunc := func(i int, instance *meshproto.NacosInstance) maputil.Entry[string, *meshresource.ServiceConsumerMetadataResource] {
		metadataJsonStr, err := convertor.ToJson(instance.Metadata)
		if err != nil {
			logger.Errorf("skipping convert service consumer %s:%d of %s to ServiceConsumerMetadata "+
				"because it cannot convert to valid json string", instance.Ip, instance.Port, serviceRes.Name)
			return maputil.Entry[string, *meshresource.ServiceConsumerMetadataResource]{}
		}
		consumerAppName, exists := instance.Metadata[constants.Application]
		if !exists {
			logger.Errorf("service consumer %s:%d of %s is invalid because no application name found, raw metadata: %s",
				instance.Ip, instance.Port, serviceRes.Name, metadataJsonStr)
			return maputil.Entry[string, *meshresource.ServiceConsumerMetadataResource]{}
		}
		serviceName, exists := instance.Metadata[constants.InterfaceKey]
		if !exists {
			logger.Errorf("service consumer %s:%d of %s is invalid because no service name found, raw metadata: %s",
				instance.Ip, instance.Port, serviceRes.Name, metadataJsonStr)
			return maputil.Entry[string, *meshresource.ServiceConsumerMetadataResource]{}
		}
		version := instance.Metadata[constants.VersionKey]
		group := instance.Metadata[constants.GroupKey]
		resKey := consumerAppName + ":" + serviceRes.Name
		res := meshresource.NewServiceConsumerMetadataResourceWithAttributes(resKey, serviceRes.Mesh)
		res.Spec = &meshproto.ServiceConsumerMetadata{
			ServiceName:     serviceName,
			ConsumerAppName: consumerAppName,
			Version:         version,
			Group:           group,
			Metadata:        instance.Metadata,
		}
		return maputil.Entry[string, *meshresource.ServiceConsumerMetadataResource]{
			Key:   res.ResourceKey(),
			Value: res,
		}
	}
	newConsumers := maputil.FromEntries(slice.Map(serviceRes.Spec.Instances, convertFunc))
	st, err := n.storeRouter.ResourceKindRoute(meshresource.ServiceConsumerMetadataKind)
	if err != nil {
		logger.Errorf("process service consumer metadata upsert event, but cannot route to service consumer metadata resource, cause: %v", err)
		return err
	}
	resources, err := st.ListByIndexes(map[string]string{
		index.ByServiceConsumerServiceName: serviceName,
	})
	if err != nil {
		logger.Errorf("process service consumer metadata upsert event, but cannot list service consumer metadata resource of %s, cause: %v", serviceRes.Name, err)
		return err
	}
	oldConsumers := make(map[string]*meshresource.ServiceConsumerMetadataResource)
	for _, res := range resources {
		oldConsumer, ok := res.(*meshresource.ServiceConsumerMetadataResource)
		if !ok {
			return bizerror.NewAssertionError(meshresource.ServiceConsumerMetadataKind, reflect.TypeOf(res).Name())
		}
		oldConsumers[oldConsumer.ResourceKey()] = oldConsumer
	}
	// Find offline consumers and delete them
	offlineConsumers := maputil.Minus(oldConsumers, newConsumers)
	slice.ForEach(maputil.Values(offlineConsumers), func(_ int, item *meshresource.ServiceConsumerMetadataResource) {
		err := st.Delete(item)
		if err != nil {
			logger.Errorf("delete service consumer metadata %s failed, cause: %v", item.ResourceKey(), err)
			return
		}
		n.emitter.Send(events.NewResourceChangedEvent(cache.Deleted, item, nil))
	})
	// Find consumers need to add and add them
	addConsumers := maputil.Minus(newConsumers, oldConsumers)
	slice.ForEach(maputil.Values(addConsumers), func(_ int, item *meshresource.ServiceConsumerMetadataResource) {
		err := st.Add(item)
		if err != nil {
			logger.Errorf("add service consumer metadata %s failed, cause: %v", item.ResourceKey(), err)
			return
		}
		n.emitter.Send(events.NewResourceChangedEvent(cache.Added, nil, item))
	})
	// Find consumers need to update and update them
	updateConsumers := maputil.Intersect(newConsumers, oldConsumers)
	slice.ForEach(maputil.Values(updateConsumers), func(_ int, item *meshresource.ServiceConsumerMetadataResource) {
		err := st.Update(item)
		if err != nil {
			logger.Errorf("update service consumer metadata %s failed, cause: %v", item.ResourceKey(), err)
			return
		}
		n.emitter.Send(events.NewResourceChangedEvent(cache.Updated, item, item))
	})

	return nil
}

func parseServiceName(s string) (string, error) {
	const prefix = "consumers:"
	if !strings.HasPrefix(s, prefix) {
		return "", fmt.Errorf("invalid prefix")
	}

	parts := strings.SplitN(s[len(prefix):], ":", 3)

	if len(parts) < 1 {
		return "", fmt.Errorf("invalid format")
	}

	return parts[0], nil
}

func (n *NacosServiceEventSubscriber) processRPCInstanceUpsert(serviceRes *meshresource.NacosServiceResource) error {
	convertFunc := func(i int, instance *meshproto.NacosInstance) maputil.Entry[string, *meshresource.RPCInstanceResource] {
		resName := meshresource.BuildInstanceResName(serviceRes.Name, instance.Ip, instance.Port)
		res := meshresource.NewRPCInstanceResourceWithAttributes(resName, serviceRes.Mesh)
		var registerTime string
		timestamp, err := strconv.ParseInt(instance.Metadata[constants.TimestampKey], 10, 64)
		if err == nil {
			registerTime = time.UnixMilli(timestamp).Format(constants.TimeFormatStr)
		}
		revision := instance.Metadata[constants.MetadataRevisionKey]
		metadataStorageType := instance.Metadata[constants.MetadataStorageTypeKey]
		urlParams, exists := instance.Metadata[constants.URLParamsKey]
		releaseVersion := ""
		protocol := ""
		serialization := ""
		preferSerialization := ""
		if exists {
			paramsMap := make(map[string]string)
			err := json.Unmarshal([]byte(urlParams), &paramsMap)
			if err != nil {
				logger.Warnf("parse url params failed, raw url params string: %s, cause: %v", urlParams, err)
			}
			releaseVersion = paramsMap[constants.ReleaseKey]
			protocol = paramsMap[constants.DubboVersionKey]
			serialization = paramsMap[constants.SerializationKey]
			preferSerialization = paramsMap[constants.SerializationKey]

		}
		var endpoints []*meshproto.Endpoint
		err = json.Unmarshal([]byte(instance.Metadata[constants.EndpointsKey]), &endpoints)
		if err != nil {
			logger.Warnf("parse endpoints failed, raw endpoints string: %s, cause: %v", instance.Metadata[constants.EndpointsKey], err)
		}
		res.Spec = &meshproto.RPCInstance{
			Name:                resName,
			AppName:             serviceRes.Name,
			Ip:                  instance.Ip,
			Port:                instance.Port,
			RegisterTime:        registerTime,
			UnregisterTime:      "",
			Revision:            revision,
			MetadataStorageType: metadataStorageType,
			ReleaseVersion:      releaseVersion,
			Protocol:            protocol,
			Serialization:       serialization,
			PreferSerialization: preferSerialization,
			Tags:                getRPCInstanceTags(instance.Metadata),
			Endpoints:           endpoints,
			Metadata:            instance.Metadata,
		}
		return maputil.Entry[string, *meshresource.RPCInstanceResource]{
			Key:   res.ResourceKey(),
			Value: res,
		}
	}
	newInstances := maputil.FromEntries(slice.Map(serviceRes.Spec.Instances, convertFunc))
	st, err := n.storeRouter.ResourceKindRoute(meshresource.RPCInstanceKind)
	if err != nil {
		logger.Errorf("process rpc instance upsert event, but cannot route to rpc instance resource, cause: %v", err)
		return err
	}
	resources, err := st.ListByIndexes(map[string]string{
		index.ByRPCInstanceAppName: serviceRes.Name,
	})
	if err != nil {
		logger.Errorf("process rpc instance upsert event, but cannot list rpc instance resource of %s, cause: %v", serviceRes.Name, err)
		return err
	}
	oldInstances := make(map[string]*meshresource.RPCInstanceResource)
	for _, res := range resources {
		oldInstance, ok := res.(*meshresource.RPCInstanceResource)
		if !ok {
			return bizerror.NewAssertionError(meshresource.RPCInstanceKind, reflect.TypeOf(res).Name())
		}
		oldInstances[oldInstance.ResourceKey()] = oldInstance
	}
	// find offline instances and delete them
	offlineInstances := maputil.Minus(oldInstances, newInstances)
	slice.ForEach(maputil.Values(offlineInstances), func(_ int, item *meshresource.RPCInstanceResource) {
		err := st.Delete(item)
		if err != nil {
			logger.Errorf("delete rpc instance %s failed, cause: %v", item.ResourceKey(), err)
			return
		}
		n.emitter.Send(events.NewResourceChangedEvent(cache.Deleted, item, nil))
	})
	// find instances need to add and add them
	addInstances := maputil.Minus(newInstances, oldInstances)
	slice.ForEach(maputil.Values(addInstances), func(_ int, item *meshresource.RPCInstanceResource) {
		err := st.Add(item)
		if err != nil {
			logger.Errorf("add rpc instance %s failed, cause: %v", item.ResourceKey(), err)
			return
		}
		n.emitter.Send(events.NewResourceChangedEvent(cache.Added, nil, item))
	})
	// find instances need to update and update them
	updateInstances := maputil.Intersect(newInstances, oldInstances)
	slice.ForEach(maputil.Values(updateInstances), func(_ int, item *meshresource.RPCInstanceResource) {
		err := st.Update(item)
		if err != nil {
			logger.Errorf("update rpc instance %s failed, cause: %v", item.ResourceKey(), err)
			return
		}
		n.emitter.Send(events.NewResourceChangedEvent(cache.Updated, item, item))
	})
	logger.Debugf("process rpc instance upsert event, oldInstances: %s, newInstances: %s, offlineInstances: %s, addInstances: %s, updateInstances: %s",
		maputil.Keys(oldInstances), maputil.Keys(newInstances), maputil.Keys(offlineInstances), maputil.Keys(addInstances), maputil.Keys(updateInstances))
	return nil
}

func getRPCInstanceTags(metadata map[string]string) map[string]string {
	knownKeys := set.New[string](constants.URLParamsKey, constants.EndpointsKey,
		constants.MetadataRevisionKey, constants.MetadataStorageTypeKey, constants.TimestampKey)
	return maputil.Filter(metadata, func(key string, value string) bool {
		return !knownKeys.Contain(key)
	})
}

func (n *NacosServiceEventSubscriber) processDelete(serviceRes *meshresource.NacosServiceResource) error {
	providerRe := regexp.MustCompile(`^providers:[\w.]+(?::[\w.]*:|::[\w.]*)?$`)
	consumerRe := regexp.MustCompile(`^consumers:[\w.]+(?::[\w.]*:|::[\w.]*)?$`)
	if providerRe.MatchString(serviceRes.Name) {
		logger.Infof("interface-level service ignored, service name: %s", serviceRes.Name)
		return nil
	}
	if consumerRe.MatchString(serviceRes.Name) {
		return n.processServiceConsumerDelete(serviceRes)
	}
	return n.processRPCInstanceDelete(serviceRes)
}

func (n *NacosServiceEventSubscriber) processServiceConsumerDelete(serviceRes *meshresource.NacosServiceResource) error {
	st, err := n.storeRouter.ResourceKindRoute(meshresource.ServiceConsumerMetadataKind)
	if err != nil {
		logger.Errorf("process service consumer delete event, but cannot route to service consumer metadata resource, cause: %v", err)
		return err
	}
	resources, err := st.ListByIndexes(map[string]string{
		index.ByServiceConsumerServiceName: serviceRes.Name,
	})
	if err != nil {
		logger.Errorf("process service consumer delete event, but cannot list service consumer metadata resource of %s, cause: %v", serviceRes.Name, err)
		return err
	}
	slice.ForEach(resources, func(_ int, item coremodel.Resource) {
		err := st.Delete(item)
		if err != nil {
			logger.Errorf("delete resource failed, resource: %s, cause: %v", item.String(), err)
		}
		n.emitter.Send(events.NewResourceChangedEvent(cache.Deleted, item, nil))
	})
	return nil
}

func (n *NacosServiceEventSubscriber) processRPCInstanceDelete(serviceRes *meshresource.NacosServiceResource) error {
	st, err := n.storeRouter.ResourceKindRoute(meshresource.RPCInstanceKind)
	if err != nil {
		logger.Errorf("process rpc instance delete event, but cannot route to rpc instance resource, cause: %v", err)
		return err
	}
	resources, err := st.ListByIndexes(map[string]string{
		index.ByRPCInstanceAppName: serviceRes.Name,
	})
	if err != nil {
		logger.Errorf("process rpc instance delete event, but cannot list rpc instance resource of %s, cause: %v", serviceRes.Name, err)
		return err
	}
	slice.ForEach(resources, func(_ int, item coremodel.Resource) {
		err := st.Delete(item)
		if err != nil {
			logger.Errorf("delete resource failed, resource: %s, cause: %v", item.String(), err)
		}
		n.emitter.Send(events.NewResourceChangedEvent(cache.Deleted, item, nil))
	})
	return nil
}
