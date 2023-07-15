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

package storage

import "github.com/apache/dubbo-admin/pkg/core/endpoint"

const (
	TagRoute       = "tagroute/v1beta1"
	ServiceMapping = "servicemapping/v1beta1"
	DynamicConfig  = "dynamicconfig/v1beta1"
	ConditionRoute = "conditionroute/v1beta1"
	Authorization  = "authorization/v1beta1"
	Authentication = "authentication/v1beta1"
)

type ToClient interface {
	Type() string
	Revision() int64
	Data() string
}

type Origin interface {
	Type() string
	Revision() int64
	Exact(endpoint *endpoint.Endpoint) (ToClient, error)
}
