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
	"fmt"
	"reflect"
	"strings"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

type ZKConfigEventSubscriber struct {
	emitter     events.Emitter
	storeRouter store.Router
}

func NewZKConfigEventSubscriber(eventEmitter events.Emitter, storeRouter store.Router) *ZKConfigEventSubscriber {
	return &ZKConfigEventSubscriber{
		emitter:     eventEmitter,
		storeRouter: storeRouter,
	}
}

func (z *ZKConfigEventSubscriber) ResourceKind() coremodel.ResourceKind {
	return meshresource.ZKConfigKind
}

func (z *ZKConfigEventSubscriber) Name() string {
	return "Discovery-" + z.ResourceKind().ToString()
}

func (z *ZKConfigEventSubscriber) AsyncEnabled() bool {
	return true
}

func (z *ZKConfigEventSubscriber) ProcessEvent(event events.Event) error {
	newObj, ok := event.NewObj().(*meshresource.ZKConfigResource)
	if !ok && event.NewObj() != nil {
		return bizerror.NewAssertionError(reflect.TypeOf(newObj), event.NewObj())
	}
	oldObj, ok := event.OldObj().(*meshresource.ZKConfigResource)
	if !ok && event.OldObj() != nil {
		return bizerror.NewAssertionError(reflect.TypeOf(oldObj), event.OldObj())
	}
	var processErr error
	switch event.Type() {
	case cache.Added, cache.Updated, cache.Replaced, cache.Sync:
		if newObj == nil || newObj.Spec == nil {
			errStr := "process zk config upsert event, but new obj is nil, skipped processing"
			logger.Errorf(errStr)
			return bizerror.New(bizerror.EventError, errStr)
		}
		processErr = z.processUpsert(newObj)
	case cache.Deleted:
		if oldObj == nil {
			errStr := "process zk config delete event, but old obj is nil, skipped processing"
			logger.Errorf(errStr)
			return bizerror.New(bizerror.EventError, errStr)
		}
		processErr = z.processDelete(oldObj)
	}
	if processErr != nil {
		logger.Errorf("process zk config event failed, cause: %s, event: %s", processErr.Error(), event.String())
		return processErr
	}
	logger.Infof("process zk config event successfully, event: %s", event.String())
	return nil
}

func (z *ZKConfigEventSubscriber) processUpsert(configRes *meshresource.ZKConfigResource) error {
	parts := strings.Split(configRes.Spec.NodeName, constants.DotSeparator)
	suffix := parts[len(parts)-1]
	if !constants.RuleSuffixSet.Contain(suffix) {
		logger.Warnf("node name is not end with rule suffix, skipped processing, nodeName: %s", configRes.Spec.NodeName)
		return nil
	}
	logger.Debugf("process zk config upsert event, config res: %s", configRes.String())
	switch suffix {
	case constants.TagRuleSuffix:
		return processConfigUpsert[*meshresource.TagRouteResource](
			configRes, meshresource.ToTagRouteResource, z.storeRouter, z.emitter)
	case constants.ConditionRuleSuffix:
		return processConfigUpsert[*meshresource.ConditionRouteResource](
			configRes, meshresource.ToConditionRouteResource, z.storeRouter, z.emitter)
	case constants.ConfiguratorsSuffix:
		return processConfigUpsert[*meshresource.DynamicConfigResource](
			configRes, meshresource.ToDynamicConfigResource, z.storeRouter, z.emitter)
	default:
		return bizerror.New(bizerror.UnknownError,
			fmt.Sprintf("unknown rule type in mesh %s, skipped processing, path: %s, raw content: %s",
				configRes.Mesh, configRes.Spec.NodeName, configRes.Spec.NodeData))
	}
}

func (z *ZKConfigEventSubscriber) processDelete(configRes *meshresource.ZKConfigResource) error {
	parts := strings.Split(configRes.Spec.NodeName, constants.DotSeparator)
	suffix := parts[len(parts)-1]
	if !constants.RuleSuffixSet.Contain(suffix) {
		logger.Warnf("node name is not end with rule suffix, skipped processing, nodeName: %s", configRes.Spec.NodeName)
		return nil
	}
	logger.Debugf("process zk config delete event, config res: %s", configRes.String())
	switch suffix {
	case constants.TagRuleSuffix:
		return processConfigDelete[*meshresource.TagRouteResource](
			configRes, meshresource.ToTagRouteResource, z.storeRouter, z.emitter)
	case constants.ConditionRuleSuffix:
		return processConfigDelete[*meshresource.ConditionRouteResource](
			configRes, meshresource.ToConditionRouteResource, z.storeRouter, z.emitter)
	case constants.ConfiguratorsSuffix:
		return processConfigDelete[*meshresource.DynamicConfigResource](
			configRes, meshresource.ToDynamicConfigResource, z.storeRouter, z.emitter)
	default:
		return bizerror.New(bizerror.UnknownError,
			fmt.Sprintf("unknown rule type in mesh %s, skipped processing, node: %s",
				configRes.Mesh, configRes.Spec.NodeName))
	}
}

func processConfigUpsert[T coremodel.Resource](
	configRes *meshresource.ZKConfigResource,
	toRuleRes meshresource.ToRuleResourceFunc,
	router store.Router,
	emitter events.Emitter) error {
	newRuleRes := toRuleRes(configRes.Mesh, configRes.Name, configRes.Spec.NodeData)
	if newRuleRes == nil {
		logger.Errorf("cannot unmarshal config in zk %s, raw content: %s", configRes.Mesh, configRes.Spec.NodeData)
		return bizerror.New(bizerror.ZKError, "cannot unmarshal rule")
	}
	st, err := router.ResourceKindRoute(newRuleRes.ResourceKind())
	if err != nil {
		logger.Errorf("get %s store failed, cause: %s", newRuleRes.ResourceKind(), err.Error())
		return err
	}
	oldRes, exists, err := st.GetByKey(newRuleRes.ResourceKey())
	if err != nil {
		logger.Errorf("get rule %s from store failed, cause: %s", newRuleRes.ResourceKey(), err.Error())
		return err
	}
	if !exists {
		err := st.Add(newRuleRes)
		if err != nil {
			logger.Errorf("add rule %s to store failed, cause: %s", newRuleRes.ResourceKey(), err.Error())
			return err
		}
		emitter.Send(events.NewResourceChangedEvent(cache.Added, nil, newRuleRes))
		return nil
	}

	err = st.Update(newRuleRes)
	if err != nil {
		logger.Errorf("update rule %s to store failed, cause: %s", newRuleRes.ResourceKey(), err.Error())
		return err
	}

	var oldMetadataRes T
	oldMetadataRes, ok := oldRes.(T)
	if !ok {
		logger.Errorf("cannot convert old rule %s to %T", newRuleRes.ResourceKey(), oldMetadataRes)
		return bizerror.NewAssertionError(reflect.TypeOf(oldMetadataRes), oldRes)
	}

	emitter.Send(events.NewResourceChangedEvent(cache.Updated, oldMetadataRes, newRuleRes))
	return nil
}

func processConfigDelete[T coremodel.Resource](
	configRes *meshresource.ZKConfigResource,
	toRuleRes meshresource.ToRuleResourceFunc,
	router store.Router,
	emitter events.Emitter) error {
	ruleRes := toRuleRes(configRes.Mesh, configRes.Name, configRes.Spec.NodeData)
	st, err := router.ResourceKindRoute(ruleRes.ResourceKind())
	if err != nil {
		logger.Errorf("get %s store failed, cause: %s", ruleRes.ResourceKind(), err.Error())
		return err
	}
	oldRes, exists, err := st.GetByKey(ruleRes.ResourceKey())
	if err != nil {
		logger.Errorf("get rule %s from store failed, cause: %s", ruleRes.ResourceKey(), err.Error())
		return err
	}
	if !exists {
		logger.Infof("rule %s not exists in store, skipped deleting", ruleRes.ResourceKey())
		return nil
	}
	oldRuleRes, ok := oldRes.(T)
	if !ok {
		return bizerror.NewAssertionError(reflect.TypeOf(oldRuleRes), oldRes)
	}
	err = st.Delete(ruleRes)
	if err != nil {
		logger.Errorf("delete rule %s from store failed, cause: %s", ruleRes.ResourceKey(), err.Error())
		return err
	}
	emitter.Send(events.NewResourceChangedEvent(cache.Deleted, oldRuleRes, nil))
	return nil
}
