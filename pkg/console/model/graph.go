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

import (
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"

	"github.com/apache/dubbo-admin/pkg/common/constants"
)

// GraphNode represents a node in the graph for AntV G6
type GraphNode struct {
	ID    string      `json:"id"`
	Label string      `json:"label"`
	Type  string      `json:"type"` // "application" or "service"
	Rule  string      `json:"rule"` // "provider", "consumer", or ""
	Data  interface{} `json:"data,omitempty"`
}

// GraphEdge represents an edge in the graph for AntV G6
type GraphEdge struct {
	Source string                 `json:"source"`
	Target string                 `json:"target"`
	Data   map[string]interface{} `json:"data,omitempty"` // Additional data for the edge
}

// GraphData represents the complete graph structure for AntV G6
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// CrossNode represents a node in the cross-linked list structure
type CrossNode struct {
	Instance *meshresource.InstanceResource
	Next     *CrossNode // pointer to next node in the same row
	Down     *CrossNode // pointer to next node in the same column
}

// CrossLinkedListGraph represents the cross-linked list structure as a directed graph
type CrossLinkedListGraph struct {
	Head *CrossNode
	Rows int // number of rows
	Cols int // number of columns
}

// ServiceGraphReq represents the request parameters for fetching the service graph
type ServiceGraphReq struct {
	Mesh        string `json:"mesh" form:"mesh" binding:"required"`
	ServiceName string `json:"serviceName" form:"serviceName" binding:"required"`
	Version     string `json:"version" form:"version"`
	Group       string `json:"group" form:"group"`
}

// ServiceKey returns the unique service identifier
func (s *ServiceGraphReq) ServiceKey() string {
	return s.ServiceName + constants.ColonSeparator + s.Version + constants.ColonSeparator + s.Group
}
