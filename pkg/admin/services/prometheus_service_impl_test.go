// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.package services

package services

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"testing"
)

func TestPrometheusServiceImpl_FlowMetrics(t *testing.T) {
	GaugeApiQPS := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "qps",
		Help: "qps",
	})
	GaugeApiQPS.Set(555)

	GaugeApiHttpTotalCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_total_count",
		Help: "Total number of HTTP requests",
	})
	GaugeApiHttpTotalCount.Set(100)

	GaugeApiHttpSuccessCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_success_count",
		Help: "Total number of HTTP success",
	})
	GaugeApiHttpSuccessCount.Set(50)

	GauApiHttpOutOfTimeExceptionCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_outOfTime_count",
		Help: "Total number of HTTP outOfTime",
	})
	GauApiHttpOutOfTimeExceptionCount.Set(10)

	GauApiHttpNotFoundCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_404_count",
		Help: "Total number of HTTP 404",
	})
	GauApiHttpNotFoundCount.Set(15)

	GauApiHttpOtherExceptionCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_other_count",
		Help: "Total number of HTTP other exception",
	})
	GauApiHttpOtherExceptionCount.Set(25)

	prometheus.MustRegister(GaugeApiQPS, GaugeApiHttpTotalCount, GaugeApiHttpSuccessCount,
		GauApiHttpOutOfTimeExceptionCount, GauApiHttpNotFoundCount, GauApiHttpOtherExceptionCount)

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":5555", nil)
}

func TestPrometheusServiceImpl_ClusterMetrics(t *testing.T) {

}
