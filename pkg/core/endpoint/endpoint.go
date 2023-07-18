// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package endpoint

import (
	"encoding/json"

	"github.com/apache/dubbo-admin/pkg/core/logger"
)

type Endpoint struct {
	ID            string         `json:"id,omitempty"`
	Ips           []string       `json:"ips,omitempty"`
	SpiffeID      string         `json:"spiffeID,omitempty"`
	KubernetesEnv *KubernetesEnv `json:"kubernetesEnv,omitempty"`
}

func (e *Endpoint) ToString() string {
	j, err := json.Marshal(e)
	if err != nil {
		logger.Sugar().Warnf("failed to marshal endpoint: %v", err)
		return ""
	}
	return string(j)
}

type KubernetesEnv struct {
	Namespace       string            `json:"namespace,omitempty"`
	DeploymentName  string            `json:"deploymentName,omitempty"`
	StatefulSetName string            `json:"statefulSetName,omitempty"`
	PodName         string            `json:"podName,omitempty"`
	PodLabels       map[string]string `json:"podLabels,omitempty"`
	PodAnnotations  map[string]string `json:"podAnnotations,omitempty"`
}
