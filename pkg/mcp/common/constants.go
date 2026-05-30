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

package common

const (
	// DefaultPageSize 默认分页大小
	DefaultPageSize = 10
	// DefaultPageNumber 默认页码
	DefaultPageNumber = 1
	// MaxDistributionLimit 服务分布查询的最大数量
	MaxDistributionLimit = 100
)

// SearchType 搜索类型枚举
type SearchType string

const (
	SearchTypeIP           SearchType = "ip"
	SearchTypeInstanceName SearchType = "instanceName"
	SearchTypeAppName      SearchType = "appName"
	SearchTypeName         SearchType = "serviceName"
)

// ServiceSide 服务端类型
type ServiceSide string

const (
	ServiceSideProvider ServiceSide = "provider"
	ServiceSideConsumer ServiceSide = "consumer"
)
