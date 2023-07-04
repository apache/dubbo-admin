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

type ConditionRouteDto struct {
	Base

	Conditions []string `json:"conditions" yaml:"conditions" binding:"required"`

	Priority      int    `json:"priority" yaml:"priority"`
	Enabled       bool   `json:"enabled" yaml:"enabled" binding:"required"`
	Force         bool   `json:"force" yaml:"force"`
	Runtime       bool   `json:"runtime" yaml:"runtime"`
	ConfigVersion string `json:"configVersion" yaml:"configVersion" binding:"required"`
}

type ConditionRoute struct {
	Priority      int      `json:"priority" yaml:"priority,omitempty"`
	Enabled       bool     `json:"enabled" yaml:"enabled"`
	Force         bool     `json:"force" yaml:"force"`
	Runtime       bool     `json:"runtime" yaml:"runtime,omitempty"`
	Key           string   `json:"key" yaml:"key"`
	Scope         string   `json:"scope" yaml:"scope"`
	Conditions    []string `json:"conditions" yaml:"conditions"`
	ConfigVersion string   `json:"configVersion" yaml:"configVersion"`
}
