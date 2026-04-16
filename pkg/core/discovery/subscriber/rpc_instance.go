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
	"reflect"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	enginecfg "github.com/apache/dubbo-admin/pkg/config/engine"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

type RPCInstanceEventSubscriber struct {
	instanceStore   store.ResourceStore
	rtInstanceStore store.ResourceStore
	eventEmitter    events.Emitter
	engineCfg       *enginecfg.Config
}

func NewRPCInstanceEventSubscriber(
	instanceStore store.ResourceStore,
	rtInstanceStore store.ResourceStore,
	emitter events.Emitter,
	engineCfg *enginecfg.Config) *RPCInstanceEventSubscriber {
	return &RPCInstanceEventSubscriber{
		instanceStore:   instanceStore,
		rtInstanceStore: rtInstanceStore,
		eventEmitter:    emitter,
		engineCfg:       engineCfg,
	}
}

func (s *RPCInstanceEventSubscriber) Name() string {
	return "Discovery-" + s.ResourceKind().ToString()
}

func (s *RPCInstanceEventSubscriber) ResourceKind() coremodel.ResourceKind {
	return meshresource.RPCInstanceKind
}

func (s *RPCInstanceEventSubscriber) AsyncEnabled() bool {
	return true
}

func (s *RPCInstanceEventSubscriber) ProcessEvent(event events.Event) error {
	newObj, ok := event.NewObj().(*meshresource.RPCInstanceResource)
	if !ok && event.NewObj() != nil {
		return bizerror.NewAssertionError(meshresource.RPCInstanceKind, event.NewObj())
	}
	oldObj, ok := event.OldObj().(*meshresource.RPCInstanceResource)
	if !ok && event.OldObj() != nil {
		return bizerror.NewAssertionError(meshresource.RPCInstanceKind, event.OldObj())
	}
	var processErr error
	switch event.Type() {
	case cache.Added, cache.Updated, cache.Replaced, cache.Sync:
		if newObj == nil {
			errStr := "process rpc instance upsert event, but new obj is nil, skipped processing"
			logger.Errorf(errStr)
			return bizerror.New(bizerror.EventError, errStr)
		}
		processErr = s.processUpsert(newObj)
	case cache.Deleted:
		if oldObj == nil {
			errStr := "process rpc instance delete event, but old obj is nil, skipped processing"
			logger.Errorf(errStr)
			return bizerror.New(bizerror.EventError, errStr)
		}
		processErr = s.processDelete(oldObj)
	}
	if processErr != nil {
		logger.Errorf("process rpc instance event failed, cause: %s, event: %s", processErr.Error(), event.String())
		return processErr
	}
	logger.Infof("process rpc instance event successfully, event: %s", event.String())
	return nil
}

func (s *RPCInstanceEventSubscriber) processUpsert(rpcInstanceRes *meshresource.RPCInstanceResource) error {
	// If the conditions follow are not met, we cannot identify it as a dubbo instance, so we skip it
	if !s.checkAttributeEnough(rpcInstanceRes) {
		logger.Warnf("cannot identify rpc instance %s as a dubbo instance, skipped updating instance", rpcInstanceRes.Name)
		return nil
	}
	instanceRes, err := s.getRelatedInstanceRes(rpcInstanceRes)
	if err != nil {
		return err
	}
	// if instance resource exists, that is to say the runtime instance exists and has been watched by engine
	// We should merge the rpc info into it
	if instanceRes != nil {
		meshresource.MergeRPCInstanceIntoInstance(rpcInstanceRes, instanceRes)
		if err := s.instanceStore.Update(instanceRes); err != nil {
			return err
		}
		logger.Infof("instance lifecycle merged rpc source, instance: %s, rpc: %s, registerState: registered, deployState: %s",
			instanceRes.ResourceKey(), rpcInstanceRes.ResourceKey(), instanceRes.Spec.DeployState)
		return nil
	}
	// Otherwise we can create a new instance resource by rpc instance
	instanceRes = meshresource.FromRPCInstance(rpcInstanceRes)
	s.findRelatedRuntimeInstanceAndMerge(instanceRes)
	if err := s.instanceStore.Add(instanceRes); err != nil {
		logger.Errorf("add instance resource failed, instance: %s, err: %s", instanceRes.ResourceKey(), err.Error())
		return err
	}
	logger.Infof("instance lifecycle created rpc-only instance, instance: %s, rpc: %s, registerState: registered",
		instanceRes.ResourceKey(), rpcInstanceRes.ResourceKey())

	instanceAddEvent := events.NewResourceChangedEvent(cache.Added, nil, instanceRes)
	s.eventEmitter.Send(instanceAddEvent)
	logger.Debugf("rpc instance upsert trigger instance add event, event: %s", instanceAddEvent.String())
	return nil
}

func (s *RPCInstanceEventSubscriber) processDelete(rpcInstanceRes *meshresource.RPCInstanceResource) error {
	instanceRes, err := s.getRelatedInstanceRes(rpcInstanceRes)
	if err != nil {
		return err
	}
	if instanceRes == nil {
		logger.Warnf("cannot find instance resource for rpc instance %s, skipped deleting instance", rpcInstanceRes.Name)
		return nil
	}
	meshresource.ClearRPCInstanceFromInstance(instanceRes)
	if meshresource.HasRuntimeInstanceSource(instanceRes) {
		if err := s.instanceStore.Update(instanceRes); err != nil {
			logger.Errorf("update instance resource failed after rpc delete, instance: %s, err: %s",
				instanceRes.ResourceKey(), err.Error())
			return err
		}
		logger.Infof("instance lifecycle rpc source removed, keep instance by runtime source, instance: %s, rpc: %s, deployState: %s",
			instanceRes.ResourceKey(), rpcInstanceRes.ResourceKey(), instanceRes.Spec.DeployState)
		instanceUpdateEvent := events.NewResourceChangedEvent(cache.Updated, instanceRes, instanceRes)
		s.eventEmitter.Send(instanceUpdateEvent)
		logger.Debugf("rpc instance delete trigger instance update event, event: %s", instanceUpdateEvent.String())
		return nil
	}
	if err := s.instanceStore.Delete(instanceRes); err != nil {
		logger.Errorf("delete instance resource failed, instance: %s, err: %s", instanceRes.ResourceKey(), err.Error())
		return err
	}
	logger.Infof("instance lifecycle rpc source removed and no runtime source remains, deleted instance: %s, rpc: %s",
		instanceRes.ResourceKey(), rpcInstanceRes.ResourceKey())
	instanceDeleteEvent := events.NewResourceChangedEvent(cache.Deleted, instanceRes, nil)
	s.eventEmitter.Send(instanceDeleteEvent)
	logger.Debugf("rpc instance delete trigger instance delete event, event: %s", instanceDeleteEvent.String())
	return nil
}

func (s *RPCInstanceEventSubscriber) checkAttributeEnough(rpcInstanceRes *meshresource.RPCInstanceResource) bool {
	if rpcInstanceRes.Spec == nil || strutil.IsBlank(rpcInstanceRes.Spec.AppName) ||
		strutil.IsBlank(rpcInstanceRes.Spec.Ip) || rpcInstanceRes.Spec.Port <= 0 ||
		strutil.IsBlank(rpcInstanceRes.Mesh) || constants.DefaultMesh == rpcInstanceRes.Mesh {
		return false
	}
	return true
}

func (s *RPCInstanceEventSubscriber) getRelatedInstanceRes(
	rpcInstanceRes *meshresource.RPCInstanceResource) (*meshresource.InstanceResource, error) {
	res, exists, err := s.instanceStore.GetByKey(coremodel.BuildResourceKey(rpcInstanceRes.Mesh, rpcInstanceRes.Name))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	instanceRes, ok := res.(*meshresource.InstanceResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.InstanceKind, reflect.TypeOf(res).Name())
	}
	return instanceRes, nil
}

func (s *RPCInstanceEventSubscriber) findRelatedRuntimeInstanceAndMerge(instanceRes *meshresource.InstanceResource) {
	switch s.engineCfg.Type {
	case enginecfg.Kubernetes:
		rtInstance := s.getRuntimeInstanceForInstance(instanceRes)
		if rtInstance == nil {
			logger.Warnf("cannot find runtime instance for instace %s, skipping merging", instanceRes.ResourceKey())
			return
		}
		meshresource.MergeRuntimeInstanceIntoInstance(rtInstance, instanceRes)
	default:
		return
	}
}

func (s *RPCInstanceEventSubscriber) getRuntimeInstanceForInstance(instanceRes *meshresource.InstanceResource) *meshresource.RuntimeInstanceResource {
	ip := instanceRes.Spec.Ip
	resources, err := s.rtInstanceStore.ListByIndexes([]index.IndexCondition{
		{IndexName: index.ByRuntimeInstanceIPIndex, Value: ip, Operator: index.Equals},
	})
	if err != nil {
		logger.Errorf("list runtime instance by ip index failed, ip: %s, err: %s", ip, err.Error())
		return nil
	}
	if len(resources) == 0 {
		return nil
	}
	candidates := make([]*meshresource.RuntimeInstanceResource, 0, len(resources))
	for _, item := range resources {
		res, ok := item.(*meshresource.RuntimeInstanceResource)
		if !ok {
			continue
		}
		candidates = append(candidates, res)
	}
	// Prefer exact runtime identity match by app + rpcPort + mesh.
	for _, candidate := range candidates {
		if candidate.Spec == nil {
			continue
		}
		if candidate.Mesh == instanceRes.Mesh &&
			candidate.Spec.AppName == instanceRes.Spec.AppName &&
			candidate.Spec.RpcPort == instanceRes.Spec.RpcPort {
			return candidate
		}
	}
	if len(candidates) > 1 {
		keys := slice.Map(candidates, func(_ int, item *meshresource.RuntimeInstanceResource) string {
			return item.ResourceKey()
		})
		logger.Warnf("multiple runtime instances share same ip %s, fallback to first candidate, runtime keys: %v, target instance: %s",
			ip, keys, instanceRes.ResourceKey())
	}
	return candidates[0]
}
