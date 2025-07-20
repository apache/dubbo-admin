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

package constants

// Registry Constants
const (
	RegistryKey                  = "registry"
	RegistryClusterKey           = "REGISTRY_CLUSTER"
	RegisterModeKey              = "register-mode"
	RegistryClusterTypeKey       = "registry-cluster-type"
	RemoteClientNameKey          = "remote-client-name"
	DefaultRegisterModeInterface = "interface"
	DefaultRegisterModeInstance  = "instance"
	DefaultRegisterModeAll       = "all"
)

const (
	SerializationKey = "prefer.serialization"
)

const (
	DubboVersionKey = "dubbo"
	WorkLoadKey     = "workLoad"
	ReleaseKey      = "release"
)

const (
	ServiceInfoSide = "side"
	ProviderSide    = "provider"
	ConsumerSide    = "consumer"
)

const (
	RetriesKey = "retries"
	TimeoutKey = "timeout"
)

const (
	Application = "application"
	Instance    = "instance"
	Service     = "service"
)

const (
	Stateful  = "有状态"
	Stateless = "无状态"
)
