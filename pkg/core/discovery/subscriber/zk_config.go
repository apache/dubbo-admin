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

// sourceRegistryZookeeper labels rule events coming from ZooKeeper so rule
// history can attribute upstream writes to author "system:zookeeper". Other
// registry subscribers (Nacos, Apollo, ...) should emit the equivalent
// SourceRegistryContextKey on their ResourceChangedEvents; until they do, rule
// history falls back to "system:upstream" for those sources.
const sourceRegistryZookeeper = "zookeeper"

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
	case constants.AffinityRuleSuffix:
		return processConfigUpsert[*meshresource.AffinityRouteResource](
			configRes, meshresource.ToAffinityRouteResource, z.storeRouter, z.emitter)
	case constants.ScriptRuleSuffix:
		return processConfigUpsert[*meshresource.ScriptRouteResource](
			configRes, meshresource.ToScriptRouteResource, z.storeRouter, z.emitter)
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
			configRes, meshresource.TagRouteKind, z.storeRouter, z.emitter)
	case constants.ConditionRuleSuffix:
		return processConfigDelete[*meshresource.ConditionRouteResource](
			configRes, meshresource.ConditionRouteKind, z.storeRouter, z.emitter)
	case constants.ConfiguratorsSuffix:
		return processConfigDelete[*meshresource.DynamicConfigResource](
			configRes, meshresource.DynamicConfigKind, z.storeRouter, z.emitter)
	case constants.AffinityRuleSuffix:
		return processConfigDelete[*meshresource.AffinityRouteResource](
			configRes, meshresource.AffinityRouteKind, z.storeRouter, z.emitter)
	case constants.ScriptRuleSuffix:
		return processConfigDelete[*meshresource.ScriptRouteResource](
			configRes, meshresource.ScriptRouteKind, z.storeRouter, z.emitter)
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
		recordConfigPlatformEvent(router, newRuleRes, "added")
		emitter.Send(events.NewResourceChangedEvent(cache.Added, nil, newRuleRes))
		emitter.Send(events.NewResourceChangedEventWithContext(cache.Added, nil, newRuleRes, map[string]string{
			events.SourceRegistryContextKey: sourceRegistryZookeeper,
		}))
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

	emitter.Send(events.NewResourceChangedEventWithContext(cache.Updated, oldMetadataRes, newRuleRes, map[string]string{
		events.SourceRegistryContextKey: sourceRegistryZookeeper,
	}))
	recordConfigPlatformEvent(router, newRuleRes, "updated")
	emitter.Send(events.NewResourceChangedEvent(cache.Updated, oldMetadataRes, newRuleRes))
	return nil
}

func processConfigDelete[T coremodel.Resource](
	configRes *meshresource.ZKConfigResource,
	ruleKind coremodel.ResourceKind,
	router store.Router,
	emitter events.Emitter) error {
	st, err := router.ResourceKindRoute(ruleKind)
	if err != nil {
		logger.Errorf("get %s store failed, cause: %s", ruleKind, err.Error())
		return err
	}
	resourceKey := coremodel.BuildResourceKey(configRes.Mesh, configRes.Name)
	oldRes, exists, err := st.GetByKey(resourceKey)
	if err != nil {
		logger.Errorf("get rule %s from store failed, cause: %s", resourceKey, err.Error())
		return err
	}
	if !exists {
		logger.Infof("rule %s not exists in store, skipped deleting", resourceKey)
		return nil
	}
	oldRuleRes, ok := oldRes.(T)
	if !ok {
		return bizerror.NewAssertionError(reflect.TypeOf(oldRuleRes), oldRes)
	}
	err = st.Delete(oldRuleRes)
	if err != nil {
		logger.Errorf("delete rule %s from store failed, cause: %s", resourceKey, err.Error())
		return err
	}
	recordConfigPlatformEvent(router, oldRuleRes, "deleted")
	emitter.Send(events.NewResourceChangedEvent(cache.Deleted, oldRuleRes, nil))
	emitter.Send(events.NewResourceChangedEventWithContext(cache.Deleted, oldRuleRes, nil, map[string]string{
		events.SourceRegistryContextKey: sourceRegistryZookeeper,
	}))
	return nil
}

func recordConfigPlatformEvent(router store.Router, res coremodel.Resource, action string) {
	serviceName, category := extractRuleEventContext(res)
	if serviceName == "" {
		return
	}
	RecordRegistryEvent(router, RegistryEventInput{
		Mesh:        res.ResourceMesh(),
		Source:      "Zookeeper",
		SourceType:  "zookeeper",
		Category:    category,
		Action:      action,
		Message:     fmt.Sprintf("Zookeeper %s %s: %s", category, action, serviceName),
		ServiceName: serviceName,
	})
}

func extractRuleEventContext(res coremodel.Resource) (string, string) {
	switch item := res.(type) {
	case *meshresource.TagRouteResource:
		if item.Spec == nil {
			return "", ""
		}
		return item.Spec.Key, "tag-route"
	case *meshresource.ConditionRouteResource:
		if item.Spec == nil {
			return "", ""
		}
		return item.Spec.Key, "condition-route"
	case *meshresource.DynamicConfigResource:
		if item.Spec == nil {
			return "", ""
		}
		return item.Spec.Key, "dynamic-config"
	case *meshresource.AffinityRouteResource:
		if item.Spec == nil {
			return "", ""
		}
		return item.Spec.Key, "affinity-route"
	case *meshresource.ScriptRouteResource:
		if item.Spec == nil {
			return "", ""
		}
		return item.Spec.Key, "script-route"
	default:
		return "", ""
	}
}
