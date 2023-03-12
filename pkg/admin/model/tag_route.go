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

type TagRouteDto struct {
	Tags []Tag `json:"tags" binding:"required"`

	Priority int  `json:"priority"`
	Enable   bool `json:"enable" binding:"required"`
	Force    bool `json:"force"`
	Runtime  bool `json:"runtime"`

	Application    string `json:"application" binding:"required"`
	Service        string `json:"service"`
	Id             string `json:"id"`
	ServiceVersion string `json:"serviceVersion" binding:"required"`
	ServiceGroup   string `json:"serviceGroup"`
}

type TagRoute struct {
	Priority int
	Enable   bool
	Force    bool
	Runtime  bool
	Key      string
	Tags     []Tag
}

type Tag struct {
	Name      string `json:"name" binding:"required"`
	Match     []MatchCondition
	Addresses []string `json:"addresses"`
}

type MatchCondition struct {
	Key   string `json:"key" binding:"required"`
	Value StringMatch
}

type StringMatch struct {
	Exact    string `json:"exact"`
	Prefix   string `json:"prefix"`
	Regex    string `json:"regex"`
	Noempty  string `json:"noempty"`
	Empty    string `json:"empty"`
	Wildcard string `json:"wildcard"`
}
