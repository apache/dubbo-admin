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

package xds

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/apache/dubbo-admin/pkg/core"
)

var statsLogger = core.Log.WithName("stats-callbacks")

const ConfigInFlightThreshold = 100_000

type StatsCallbacks interface {
	// ConfigReadyForDelivery marks a configuration as a ready to be delivered.
	// This means that any config (EDS/CDS/DDS policies etc.) with specified version was set to a Snapshot
	// and it's scheduled to be delivered.
	ConfigReadyForDelivery(configVersion string)
	// DiscardConfig removes a configuration from being delivered.
	// This should be called when the client of xDS/DDS server disconnects.
	DiscardConfig(configVersion string)
	Callbacks
	DeltaCallbacks
}

type statsCallbacks struct {
	NoopCallbacks
	responsesSentMetric    *prometheus.CounterVec
	requestsReceivedMetric *prometheus.CounterVec
	deliveryMetric         prometheus.Summary
	deliveryMetricName     string
	streamsActive          int
	configsQueue           map[string]time.Time
	sync.RWMutex
}

func (s *statsCallbacks) ConfigReadyForDelivery(configVersion string) {
	s.Lock()
	if len(s.configsQueue) > ConfigInFlightThreshold {
		// We clean up times of ready for delivery configs when config is delivered or client is disconnected.
		// However, there is always a potential case that may have missed.
		// When we get to the point of ConfigInFlightThreshold elements in the map we want to wipe the map
		// instead of grow it to the point that CP runs out of memory.
		// The statistic is not critical for CP to work, and we will still get data points of configs that are constantly being delivered.
		statsLogger.Info("cleaning up config ready for delivery times to avoid potential memory leak. This operation may cause problems with metric for a short period of time", "metric", s.deliveryMetricName)
		s.configsQueue = map[string]time.Time{}
	}
	s.configsQueue[configVersion] = core.Now()
	s.Unlock()
}

func (s *statsCallbacks) DiscardConfig(configVersion string) {
	s.Lock()
	delete(s.configsQueue, configVersion)
	s.Unlock()
}

var _ StatsCallbacks = &statsCallbacks{}

func NewStatsCallbacks(metrics prometheus.Registerer, dsType string) (StatsCallbacks, error) {
	stats := &statsCallbacks{
		configsQueue: map[string]time.Time{},
	}

	return stats, nil
}

func (s *statsCallbacks) OnStreamOpen(context.Context, int64, string) error {
	s.Lock()
	defer s.Unlock()
	s.streamsActive++
	return nil
}

func (s *statsCallbacks) OnStreamClosed(int64) {
	s.Lock()
	defer s.Unlock()
	s.streamsActive--
}

func (s *statsCallbacks) OnStreamRequest(_ int64, request DiscoveryRequest) error {
	if request.VersionInfo() == "" {
		return nil // It's initial DiscoveryRequest to ask for resources. It's neither ACK nor NACK.
	}

	if request.HasErrors() {
		s.requestsReceivedMetric.WithLabelValues(request.GetTypeUrl(), "NACK").Inc()
	} else {
		s.requestsReceivedMetric.WithLabelValues(request.GetTypeUrl(), "ACK").Inc()
	}

	if configTime, exists := s.takeConfigTimeFromQueue(request.VersionInfo()); exists {
		s.deliveryMetric.Observe(float64(core.Now().Sub(configTime).Milliseconds()))
	}
	return nil
}

func (s *statsCallbacks) takeConfigTimeFromQueue(configVersion string) (time.Time, bool) {
	s.Lock()
	generatedTime, ok := s.configsQueue[configVersion]
	delete(s.configsQueue, configVersion)
	s.Unlock()
	return generatedTime, ok
}

func (s *statsCallbacks) OnStreamResponse(_ int64, _ DiscoveryRequest, response DiscoveryResponse) {
	s.responsesSentMetric.WithLabelValues(response.GetTypeUrl()).Inc()
}

func (s *statsCallbacks) OnDeltaStreamOpen(context.Context, int64, string) error {
	s.Lock()
	defer s.Unlock()
	s.streamsActive++
	return nil
}

func (s *statsCallbacks) OnDeltaStreamClosed(int64) {
	s.Lock()
	defer s.Unlock()
	s.streamsActive--
}

func (s *statsCallbacks) OnStreamDeltaRequest(_ int64, request DeltaDiscoveryRequest) error {
	if request.GetResponseNonce() == "" {
		return nil // It's initial DiscoveryRequest to ask for resources. It's neither ACK nor NACK.
	}

	if request.HasErrors() {
		s.requestsReceivedMetric.WithLabelValues(request.GetTypeUrl(), "NACK").Inc()
	} else {
		s.requestsReceivedMetric.WithLabelValues(request.GetTypeUrl(), "ACK").Inc()
	}

	// Delta only has an initial version, therefore we need to change the key to nodeID and typeURL.
	if configTime, exists := s.takeConfigTimeFromQueue(request.NodeId() + request.GetTypeUrl()); exists {
		s.deliveryMetric.Observe(float64(core.Now().Sub(configTime).Milliseconds()))
	}
	return nil
}

func (s *statsCallbacks) OnStreamDeltaResponse(_ int64, _ DeltaDiscoveryRequest, response DeltaDiscoveryResponse) {
	s.responsesSentMetric.WithLabelValues(response.GetTypeUrl()).Inc()
}
