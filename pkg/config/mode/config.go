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

package mode

import (
	"github.com/pkg/errors"
)

// DeployMode 部署模型
// 1. 纯Kubernetes
// 2. 半托管, 使用Kubernetes做平台, 服务注册的模型仍然使用zookeeper
// 3. vm, 使用vm机器传统服务注册模型
type DeployMode = string

const (
	KubernetesMode DeployMode = "k8s"       // 全托管
	HalfHostMode   DeployMode = "half"      // 半托管
	UniversalMode  DeployMode = "universal" // vm传统
)

// Mode Control Plane mode
type Mode = string

const (
	Zone   Mode = "zone"
	Global Mode = "global"
	Test   Mode = "test"
)

// ValidateMode to check modes of dubbo-cp
func ValidateMode(mode Mode) error {
	if mode != Zone && mode != Global && mode != Test {
		return errors.Errorf("invalid mode. Available modes: %s, %s, %s", Zone, Global, Test)
	}
	return nil
}
