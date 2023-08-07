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

package kube

type KubeConfig struct {
	Namespace   string `yaml:"namespace"`
	ServiceName string `yaml:"service-name"`

	InPodEnv              bool `yaml:"in-pod-env"`
	IsKubernetesConnected bool `yaml:"is-kubernetes-connected"`
	// Qps for rest config
	RestConfigQps int `yaml:"rest-config-qps"`
	// Burst for rest config
	RestConfigBurst int `yaml:"rest-config-burst"`

	KubeFileConfig string `yaml:"kube-file-config"`

	DomainSuffix string `yaml:"domain-suffix"`
}

func (o *KubeConfig) Sanitize() {}

func (o *KubeConfig) Validate() error {
	// TODO Validate options config
	return nil
}
