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

	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/core/events"
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
	raw := counter.GetAll()
	result := make(map[string]int64, len(raw))
	for k, v := range raw {
		result[k] = v
	}
	return result
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
	}
	return nil
}

func (cm *counterManager) handleEvent(kind resmodel.ResourceKind, event events.Event) error {
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

func processSimpleCounter(counter *Counter, event events.Event) {
	switch event.Type() {
	case cache.Added:
		counter.Increment()
	case cache.Sync, cache.Replaced:
		if isNewResourceEvent(event) {
			counter.Increment()
		}
	case cache.Deleted:
		counter.Decrement()
	case cache.Updated:
		// no-op for simple counters
	default:
	}
}

func processDistributionCounter(cfg *distributionCounterConfig, event events.Event) {
	switch event.Type() {
	case cache.Added:
		cfg.increment(cfg.extractFrom(event.NewObj()))
	case cache.Sync, cache.Replaced:
		if isNewResourceEvent(event) {
			cfg.increment(cfg.extractFrom(event.NewObj()))
		} else {
			cfg.update(event.OldObj(), event.NewObj())
		}
	case cache.Updated:
		cfg.update(event.OldObj(), event.NewObj())
	case cache.Deleted:
		cfg.decrement(cfg.extractFrom(event.OldObj()))
	default:
	}
}

func (cfg *distributionCounterConfig) extractFrom(res resmodel.Resource) string {
	if cfg.extractor == nil || res == nil {
		return ""
	}
	return cfg.extractor(res)
}

func (cfg *distributionCounterConfig) increment(key string) {
	cfg.counter.Increment(normalizeDistributionKey(key))
}

func (cfg *distributionCounterConfig) decrement(key string) {
	cfg.counter.Decrement(normalizeDistributionKey(key))
}

func (cfg *distributionCounterConfig) update(oldObj, newObj resmodel.Resource) {
	oldKey := normalizeDistributionKey(cfg.extractFrom(oldObj))
	newKey := normalizeDistributionKey(cfg.extractFrom(newObj))
	if oldKey == newKey {
		return
	}
	if oldObj != nil {
		cfg.counter.Decrement(oldKey)
	}
	if newObj != nil {
		cfg.counter.Increment(newKey)
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
