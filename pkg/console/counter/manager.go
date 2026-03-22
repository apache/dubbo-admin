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

package counter

import (
	"fmt"
	"sync/atomic"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	resmodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type CounterType string

type FieldExtractor func(resmodel.Resource) string

const (
	ProtocolCounter  CounterType = "protocol"
	ReleaseCounter   CounterType = "release"
	DiscoveryCounter CounterType = "discovery"
)

type CounterManager interface {
	RegisterSimpleCounter(kind resmodel.ResourceKind)
	RegisterDistributionCounter(kind resmodel.ResourceKind, metric CounterType, extractor FieldExtractor)
	Count(kind resmodel.ResourceKind) int64
	Distribution(metric CounterType) map[string]int64
	CountByMesh(kind resmodel.ResourceKind, mesh string) int64
	DistributionByMesh(metric CounterType, mesh string) map[string]int64
	Reset()
	Bind(bus events.EventBus) error
}

type distributionCounterConfig struct {
	counterType CounterType
	counter     *DistributionCounter
	extractor   func(resmodel.Resource) string
}

type counterManager struct {
	simpleCounters      map[resmodel.ResourceKind]*Counter
	distributionConfigs map[resmodel.ResourceKind][]*distributionCounterConfig
	distributionByType  map[CounterType]*DistributionCounter
	isLeader            *atomic.Bool
}

func NewCounterManager() CounterManager {
	return newCounterManager()
}

func newCounterManager() *counterManager {
	cm := &counterManager{
		simpleCounters:      make(map[resmodel.ResourceKind]*Counter),
		distributionConfigs: make(map[resmodel.ResourceKind][]*distributionCounterConfig),
		distributionByType:  make(map[CounterType]*DistributionCounter),
	}

	cm.RegisterSimpleCounter(meshresource.ApplicationKind)
	cm.RegisterSimpleCounter(meshresource.ServiceProviderMetadataKind)
	cm.RegisterSimpleCounter(meshresource.InstanceKind)

	cm.RegisterDistributionCounter(meshresource.InstanceKind, ProtocolCounter, instanceProtocolKey)
	cm.RegisterDistributionCounter(meshresource.InstanceKind, ReleaseCounter, instanceReleaseKey)
	cm.RegisterDistributionCounter(meshresource.InstanceKind, DiscoveryCounter, instanceMeshKey)

	return cm
}

func (cm *counterManager) RegisterSimpleCounter(kind resmodel.ResourceKind) {
	if kind == "" {
		return
	}
	if _, exists := cm.simpleCounters[kind]; exists {
		return
	}
	cm.simpleCounters[kind] = NewCounter(string(kind))
}

func (cm *counterManager) RegisterDistributionCounter(kind resmodel.ResourceKind, metric CounterType, extractor FieldExtractor) {
	if kind == "" || metric == "" {
		return
	}
	counter := cm.distributionByType[metric]
	if counter == nil {
		counter = NewDistributionCounter(string(metric))
		cm.distributionByType[metric] = counter
	}

	configs := cm.distributionConfigs[kind]
	for _, cfg := range configs {
		if cfg.counterType == metric {
			cfg.counter = counter
			cfg.extractor = extractor
			return
		}
	}

	cm.distributionConfigs[kind] = append(configs, &distributionCounterConfig{
		counterType: metric,
		counter:     counter,
		extractor:   extractor,
	})
}

func (cm *counterManager) Reset() {
	for _, counter := range cm.simpleCounters {
		counter.Reset()
	}
	for _, counter := range cm.distributionByType {
		counter.Reset()
	}
}

func (cm *counterManager) Count(kind resmodel.ResourceKind) int64 {
	if counter, exists := cm.simpleCounters[kind]; exists {
		return counter.Get()
	}
	return 0
}

func (cm *counterManager) Distribution(metric CounterType) map[string]int64 {
	counter, exists := cm.distributionByType[metric]
	if !exists {
		return map[string]int64{}
	}
	return counter.GetAll()
}

func (cm *counterManager) CountByMesh(kind resmodel.ResourceKind, mesh string) int64 {
	if mesh == "" {
		mesh = "default"
	}
	if counter, exists := cm.simpleCounters[kind]; exists {
		return counter.GetByGroup(mesh)
	}
	return 0
}

func (cm *counterManager) DistributionByMesh(metric CounterType, mesh string) map[string]int64 {
	if mesh == "" {
		mesh = "default"
	}
	counter, exists := cm.distributionByType[metric]
	if !exists {
		return map[string]int64{}
	}
	return counter.GetByGroup(mesh)
}

func (cm *counterManager) Bind(bus events.EventBus) error {
	handledKinds := make(map[resmodel.ResourceKind]struct{})
	for kind := range cm.simpleCounters {
		handledKinds[kind] = struct{}{}
	}
	for kind := range cm.distributionConfigs {
		handledKinds[kind] = struct{}{}
	}

	for kind := range handledKinds {
		resourceKind := kind
		name := fmt.Sprintf("counter-manager/%s", resourceKind)
		subscriber := &counterEventSubscriber{
			kind: resourceKind,
			name: name,
			handler: func(event events.Event) error {
				return cm.handleEvent(resourceKind, event)
			},
		}
		if err := bus.Subscribe(subscriber); err != nil {
			return err
		}
		logger.Infof("CounterManager subscribed to %s events", resourceKind)
	}
	logger.Infof("CounterManager bound to EventBus successfully")
	return nil
}

func (cm *counterManager) handleEvent(kind resmodel.ResourceKind, event events.Event) error {
	if cm.isLeader != nil && !cm.isLeader.Load() {
		return nil
	}
	logger.Debugf("CounterManager handling %s event, type: %s", kind, event.Type())
	if counter := cm.simpleCounters[kind]; counter != nil {
		processSimpleCounter(counter, event)
	}
	if configs := cm.distributionConfigs[kind]; len(configs) > 0 {
		for _, cfg := range configs {
			processDistributionCounter(cfg, event)
		}
	}
	return nil
}

func (cm *counterManager) getDistributionConfig(kind resmodel.ResourceKind, metric CounterType) *distributionCounterConfig {
	configs := cm.distributionConfigs[kind]
	for _, cfg := range configs {
		if cfg.counterType == metric {
			return cfg
		}
	}
	return nil
}

func extractMeshName(res resmodel.Resource) string {
	if res == nil {
		return "default"
	}
	mesh := res.ResourceMesh()
	if mesh == "" {
		return "default"
	}
	return mesh
}

func processSimpleCounter(counter *Counter, event events.Event) {
	mesh := extractMeshName(event.NewObj())
	if event.NewObj() == nil {
		mesh = extractMeshName(event.OldObj())
	}

	switch event.Type() {
	case cache.Added:
		counter.Increment(mesh)
		logger.Debugf("CounterManager: Increment %s for mesh=%s, current count=%d", counter.name, mesh, counter.GetByGroup(mesh))
	case cache.Sync, cache.Replaced:
		if isNewResourceEvent(event) {
			counter.Increment(mesh)
			logger.Debugf("CounterManager: Increment %s for mesh=%s (Sync/Replaced), current count=%d", counter.name, mesh, counter.GetByGroup(mesh))
		}
	case cache.Deleted:
		counter.Decrement(mesh)
		logger.Debugf("CounterManager: Decrement %s for mesh=%s, current count=%d", counter.name, mesh, counter.GetByGroup(mesh))
	case cache.Updated:
	default:
	}
}

func processDistributionCounter(cfg *distributionCounterConfig, event events.Event) {
	mesh := extractMeshName(event.NewObj())
	if event.NewObj() == nil {
		mesh = extractMeshName(event.OldObj())
	}

	switch event.Type() {
	case cache.Added:
		key := cfg.extractFrom(event.NewObj())
		cfg.counter.Increment(mesh, normalizeDistributionKey(key))
	case cache.Sync, cache.Replaced:
		if isNewResourceEvent(event) {
			key := cfg.extractFrom(event.NewObj())
			cfg.counter.Increment(mesh, normalizeDistributionKey(key))
		} else {
			cfg.update(event.OldObj(), event.NewObj())
		}
	case cache.Updated:
		cfg.update(event.OldObj(), event.NewObj())
	case cache.Deleted:
		key := cfg.extractFrom(event.OldObj())
		cfg.counter.Decrement(mesh, normalizeDistributionKey(key))
	default:
	}
}

func (cfg *distributionCounterConfig) extractFrom(res resmodel.Resource) string {
	if cfg.extractor == nil || res == nil {
		return ""
	}
	return cfg.extractor(res)
}

func (cfg *distributionCounterConfig) update(oldObj, newObj resmodel.Resource) {
	oldKey := normalizeDistributionKey(cfg.extractFrom(oldObj))
	newKey := normalizeDistributionKey(cfg.extractFrom(newObj))
	oldMesh := extractMeshName(oldObj)
	newMesh := extractMeshName(newObj)
	if oldKey == newKey && oldMesh == newMesh {
		return
	}
	if oldObj != nil {
		cfg.counter.Decrement(oldMesh, oldKey)
	}
	if newObj != nil {
		cfg.counter.Increment(newMesh, newKey)
	}
}

func instanceProtocolKey(res resmodel.Resource) string {
	instance, ok := res.(*meshresource.InstanceResource)
	if !ok || instance == nil || instance.Spec == nil {
		return ""
	}
	return instance.Spec.GetProtocol()
}

func instanceReleaseKey(res resmodel.Resource) string {
	instance, ok := res.(*meshresource.InstanceResource)
	if !ok || instance == nil || instance.Spec == nil {
		return ""
	}
	return instance.Spec.GetReleaseVersion()
}

func instanceMeshKey(res resmodel.Resource) string {
	instance, ok := res.(*meshresource.InstanceResource)
	if !ok || instance == nil {
		return ""
	}
	return instance.Mesh
}

func normalizeDistributionKey(key string) string {
	if key == "" {
		return "unknown"
	}
	return key
}

func isNewResourceEvent(event events.Event) bool {
	if event == nil {
		return false
	}
	return event.OldObj() == nil
}

type counterEventSubscriber struct {
	kind    resmodel.ResourceKind
	name    string
	handler func(events.Event) error
}

func (s *counterEventSubscriber) ResourceKind() resmodel.ResourceKind {
	return s.kind
}

func (s *counterEventSubscriber) Name() string {
	return s.name
}

func (s *counterEventSubscriber) ProcessEvent(event events.Event) error {
	if s.handler == nil {
		return nil
	}
	return s.handler(event)
}
