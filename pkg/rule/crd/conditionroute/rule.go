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

package conditionroute

import (
	"encoding/json"

	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/rule/storage"
)

type ToClient struct {
	revision int64
	data     string
}

func (r *ToClient) Type() string {
	return storage.ConditionRoute
}

func (r *ToClient) Revision() int64 {
	return r.revision
}

func (r *ToClient) Data() string {
	return r.data
}

type Origin struct {
	revision int64
	data     map[string]*Policy
}

func (o *Origin) Type() string {
	return storage.ConditionRoute
}

func (o *Origin) Revision() int64 {
	return o.revision
}

func (o *Origin) Exact(endpoint *endpoint.Endpoint) (storage.ToClient, error) {
	matchedRuled := make([]*PolicyToClient, 0, len(o.data))

	for _, v := range o.data {
		if v.Spec == nil {
			continue
		}

		toClient := v.CopyToClient()
		matchedRuled = append(matchedRuled, toClient)
	}

	allRule, err := json.Marshal(matchedRuled)
	if err != nil {
		return nil, err
	}

	return &ToClient{
		revision: o.revision,
		data:     string(allRule),
	}, nil
}
