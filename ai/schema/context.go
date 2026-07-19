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

package schema

const (
	AIContextVersion  = 1
	AIContextMaxBytes = 8 * 1024
)

type AIContextGlobal struct {
	Locale string `json:"locale,omitempty"`
}

type AIContextPage struct {
	RouteName string         `json:"routeName,omitempty"`
	Path      string         `json:"path"`
	FullPath  string         `json:"fullPath,omitempty"`
	ActiveTab string         `json:"activeTab,omitempty"`
	Params    map[string]any `json:"params,omitempty"`
	Query     map[string]any `json:"query,omitempty"`
}

type AIContextScope struct {
	Mesh        string `json:"mesh"`
	Application string `json:"application,omitempty"`
	Service     string `json:"service,omitempty"`
	Instance    string `json:"instance,omitempty"`
	Rule        string `json:"rule,omitempty"`
}

type AIContextState struct {
	Filters        map[string]any `json:"filters,omitempty"`
	Selection      map[string]any `json:"selection,omitempty"`
	UnsavedChanges map[string]any `json:"unsavedChanges,omitempty"`
}

type AIContextSection struct {
	ID         string         `json:"id"`
	Source     string         `json:"source"`
	CapturedAt string         `json:"capturedAt,omitempty"`
	Priority   int            `json:"priority,omitempty"`
	Data       map[string]any `json:"data"`
}

type AIContextTruncation struct {
	Truncated       bool     `json:"truncated"`
	OmittedSections []string `json:"omittedSections"`
}

type AIContextSnapshot struct {
	Version    int                  `json:"version"`
	CapturedAt string               `json:"capturedAt"`
	Global     AIContextGlobal      `json:"global"`
	Page       AIContextPage        `json:"page"`
	Scope      AIContextScope       `json:"scope"`
	State      *AIContextState      `json:"state,omitempty"`
	Evidence   []AIContextSection   `json:"evidence,omitempty"`
	Truncation *AIContextTruncation `json:"truncation,omitempty"`
}
