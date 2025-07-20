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

package cache

import (
	"time"

	"github.com/goburrow/cache"
	"github.com/prometheus/client_golang/prometheus"
)

const ResultLabel = "result"

func NewMetric(name, help string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, []string{ResultLabel})
}

type PrometheusStatsCounter struct {
	Metric *prometheus.CounterVec
}

var _ cache.StatsCounter = &PrometheusStatsCounter{}

func (p *PrometheusStatsCounter) RecordHits(count uint64) {
	p.Metric.WithLabelValues("hit").Add(float64(count))
}

func (p *PrometheusStatsCounter) RecordMisses(count uint64) {
	p.Metric.WithLabelValues("miss").Add(float64(count))
}

func (p *PrometheusStatsCounter) RecordLoadSuccess(loadTime time.Duration) {
}

func (p *PrometheusStatsCounter) RecordLoadError(loadTime time.Duration) {
}

func (p *PrometheusStatsCounter) RecordEviction() {
}

func (p *PrometheusStatsCounter) Snapshot(stats *cache.Stats) {
}
