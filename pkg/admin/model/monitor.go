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
// limitations under the License.

package model

type Target struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

type ClusterMetricsRes struct {
	Data map[string]int `json:"data"`
}

type FlowMetricsRes struct {
	Data map[string]float64 `json:"data"`
}

type Metadata struct {
	Versions       []interface{} `json:"versions"`
	ConfigCenter   string        `json:"configCenter"`
	Registry       string        `json:"registry"`
	MetadataCenter string        `json:"metadataCenter"`
	Protocols      []interface{} `json:"protocols"`
	Rules          []string      `json:"rules"`
}
