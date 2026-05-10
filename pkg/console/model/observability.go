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

type AppDashboardReq struct {
	AppName string `form:"appName"`
	Mesh    string `form:"appName"`
}

type InstanceDashboardReq struct {
	InstanceName string `form:"instanceName"`
	Mesh         string `form:"mesh"`
}

type ServiceDashboardReq struct {
	ServiceName string `form:"serviceName"`
	Version     string `form:"version"`
	Group       string `form:"group"`
	Mesh        string `form:"mesh"`
}

type DashboardResp struct {
	FullURL string `json:"fullURL"`
}

// Metric represents a single metric with its name, labels, and value.
type Metric struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
}

type MetricsReq struct {
	InstanceName string `form:"instanceName"`
	Mesh         string `form:"mesh"`
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
