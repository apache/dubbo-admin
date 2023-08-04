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

package constant

import (
	set "github.com/dubbogo/gost/container/set"
)

const (
	DubboPropertyKey         = "dubbo.properties"
	RegistryAddressKey       = "dubbo.registry.address"
	MetadataReportAddressKey = "dubbo.metadata-report.address"
)

const (
	AnyValue               = "*"
	AnyHostValue           = "0.0.0.0"
	InterfaceKey           = "interface"
	GroupKey               = "group"
	VersionKey             = "version"
	ClassifierKey          = "classifier"
	CategoryKey            = "category"
	ProvidersCategory      = "providers"
	ConsumersCategory      = "consumers"
	RoutersCategory        = "routers"
	ConfiguratorsCategory  = "configurators"
	ConfiguratorRuleSuffix = ".configurators"
	EnabledKey             = "enabled"
	CheckKey               = "check"
	AdminProtocol          = "admin"
	Side                   = "side"
	ConsumerSide           = "consumer"
	ProviderSide           = "provider"
	ConsumerProtocol       = "consumer"
	EmptyProtocol          = "empty"
	OverrideProtocol       = "override"
	DefaultGroup           = "dubbo"
	ApplicationKey         = "application"
	DynamicKey             = "dynamic"
	SerializationKey       = "serialization"
	TimeoutKey             = "timeout"
	DefaultTimeout         = 1000
	WeightKey              = "weight"
	BalancingKey           = "balancing"
	DefaultWeight          = 100
	OwnerKey               = "owner"
	Application            = "application"
	Service                = "service"
	Colon                  = ":"
	InterrogationPoint     = "?"
	IP                     = "ip"
	PlusSigns              = "+"
	PunctuationPoint       = "."
	ConditionRoute         = "condition_route"
	TagRoute               = "tag_route"
	ConditionRuleSuffix    = ".condition-router"
	TagRuleSuffix          = ".tag-router"
	ConfigFileEnvKey       = "conf" // config file path
	RegistryAll            = "ALL"
	RegistryInterface      = "INTERFACE"
	RegistryInstance       = "INSTANCE"
	RegistryType           = "TYPE"
	NamespaceKey           = "namespace"
)

var Configs = set.NewSet(WeightKey, BalancingKey)

const (
	MetricsQps                        = "dubbo_consumer_qps_total"                                 // QPS
	MetricsHttpRequestTotalCount      = "dubbo_consumer_requests_total"                            // Total number of http requests
	MetricsHttpRequestSuccessCount    = "dubbo_consumer_requests_succeed_total"                    // Total number of http successful requests
	MetricsHttpRequestOutOfTimeCount  = "dubbo_consumer_requests_timeout_total"                    // Total number of http out of time requests
	MetricsHttpRequestAddressNotFount = "dubbo_consumer_requests_failed_service_unavailable_total" // Total number of HTTP requests where the address cannot be found
	MetricsHttpRequestOtherException  = "dubbo_consumer_requests_unknown_failed_total"             // Total number of other errors for http requests
)
