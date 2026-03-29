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
	"fmt"
	"strings"

	"github.com/apache/dubbo-admin/pkg/common/constants"
)

type GraphNode struct {
	ID    string         `json:"id"`
	Label string         `json:"label"`
	Type  string         `json:"type"`
	Rule  string         `json:"rule"`
	Data  map[string]any `json:"data,omitempty"`
}

type GraphEdge struct {
	Source string         `json:"source"`
	Target string         `json:"target"`
	Data   map[string]any `json:"data,omitempty"`
}

type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type ServiceGraphReq struct {
	Mesh        string `json:"mesh" form:"mesh" binding:"required"`
	ServiceName string `json:"serviceName" form:"serviceName"`
	Version     string `json:"version" form:"version"`
	Group       string `json:"group" form:"group"`
	ServiceKey  string `json:"serviceKey" form:"serviceKey"`
}

func (s *ServiceGraphReq) Normalize() error {
	if s.ServiceKey == "" {
		if s.ServiceName == "" {
			return fmt.Errorf("serviceName or serviceKey is required")
		}
		return nil
	}

	parts := strings.Split(s.ServiceKey, constants.ColonSeparator)
	if len(parts) != 3 {
		return fmt.Errorf("invalid serviceKey")
	}

	if s.ServiceName != "" || s.Version != "" || s.Group != "" {
		if s.ServiceName != parts[0] || s.Version != parts[1] || s.Group != parts[2] {
			return fmt.Errorf("serviceKey does not match serviceName/version/group")
		}
	}

	s.ServiceName = parts[0]
	s.Version = parts[1]
	s.Group = parts[2]
	return nil
}

func (s *ServiceGraphReq) ServiceIdentityKey() string {
	return s.ServiceName + constants.ColonSeparator + s.Version + constants.ColonSeparator + s.Group
}
