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

const (
	DubboPropertyKey         = "dubbo.properties"
	RegistryAddressKey       = "dubbo.registry.address"
	MetadataReportAddressKey = "dubbo.metadata-report.address"
)

const (
	RegistryKey                  = "registry"
	RegistryClusterKey           = "REGISTRY_CLUSTER"
	RegisterModeKey              = "register-mode"
	RegistryClusterTypeKey       = "registry-cluster-type"
	RemoteClientNameKey          = "remote-client-name"
	DefaultRegisterModeInterface = "interface"
	DefaultRegisterModeInstance  = "instance"
	DefaultRegisterModeAll       = "all"
	MetadataStorageTypeKey       = "dubbo.metadata.storage-type"
	TimestampKey                 = "timestamp"
	EndpointsKey                 = "dubbo.endpoints"
	URLParamsKey                 = "dubbo.metadata-service.url-params"
	MetadataRevisionKey          = "dubbo.metadata.revision"
	AnyValue                     = "*"
	AnyHostValue                 = "0.0.0.0"
	InterfaceKey                 = "interface"
	GroupKey                     = "group"
	VersionKey                   = "version"
	ClassifierKey                = "classifier"
	CategoryKey                  = "category"
	ProvidersCategory            = "providers"
	ConsumersCategory            = "consumers"
	RoutersCategory              = "routers"
	EnabledKey                   = "enabled"
	CheckKey                     = "check"
	AdminProtocol                = "admin"
	Side                         = "side"
	ConsumerSide                 = "consumer"
	ProviderSide                 = "provider"
	ConsumerProtocol             = "consumer"
	EmptyProtocol                = "empty"
	OverrideProtocol             = "override"
	DefaultGroup                 = "dubbo"
	DynamicKey                   = "dynamic"
	SerializationKey             = "serialization"
	PreferSerializationKey       = "prefer.serialization"
	TimeoutKey                   = "timeout"
	DefaultTimeout               = 1000
	WeightKey                    = "weight"
	BalancingKey                 = "balancing"
	DefaultWeight                = 100
	OwnerKey                     = "owner"

	ConfigFileEnvKey  = "conf" // config file path
	RegistryAll       = "ALL"
	RegistryInterface = "INTERFACE"
	RegistryInstance  = "INSTANCE"
	RegistryType      = "TYPE"
	NamespaceKey      = "namespace"
)

const (
	Application = "application"
	Instance    = "instance"
	Service     = "service"

	StatefulSet = "StatefulSet"
	Deployment  = "Deployment"
)

const (
	DubboVersionKey = "dubbo"
	WorkLoadKey     = "workLoad"
	ReleaseKey      = "release"
)

const (
	Stateful  = "有状态"
	Stateless = "无状态"
)
