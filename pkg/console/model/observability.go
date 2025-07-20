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

package model

type DashboardReq interface {
	GetKeyVariable() string
}

type AppDashboardReq struct {
	Application string `form:"application"`
}

func (req *AppDashboardReq) GetKeyVariable() string {
	return req.Application
}

type InstanceDashboardReq struct {
	Instance string `form:"instance"`
}

func (req *InstanceDashboardReq) GetKeyVariable() string {
	return req.Instance
}

type ServiceDashboardReq struct {
	Service string `form:"service"`
}

func (req *ServiceDashboardReq) GetKeyVariable() string {
	return req.Service
}

// DashboardResp TODO add dynamic variables
type DashboardResp struct {
	BaseURL string `json:"baseURL"`
}

// Metric represents a single metric with its name, labels, and value.
type Metric struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
}

type MetricsReq struct {
	InstanceName string `form:"instanceName"`
}

type MetricsResp struct {
	InstanceName string
	Metrics      []Metric
}

type MetricsCategory int

const (
	RT MetricsCategory = iota
	QPS
	REQUESTS
	APPLICATION
	CONFIGCENTER
	REGISTRY
	METADATA
	THREAD_POOL
)
