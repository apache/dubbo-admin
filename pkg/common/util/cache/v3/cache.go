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

package v3

import (
	"sort"

	v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	ctl_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	protov1 "github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func ToDeltaDiscoveryResponse(s ctl_cache.Snapshot) (*v3.DeltaDiscoveryResponse, error) {
	resp := &v3.DeltaDiscoveryResponse{}
	for _, rs := range s.Resources {
		for _, name := range sortedResourceNames(rs) {
			r := rs.Items[name]
			pbany, err := anypb.New(protov1.MessageV2(r.Resource))
			if err != nil {
				return nil, err
			}
			resp.Resources = append(resp.Resources, &v3.Resource{
				Version:  rs.Version,
				Name:     name,
				Resource: pbany,
			})
		}
	}
	return resp, nil
}

func sortedResourceNames(rs ctl_cache.Resources) []string {
	names := make([]string, 0, len(rs.Items))
	for name := range rs.Items {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
