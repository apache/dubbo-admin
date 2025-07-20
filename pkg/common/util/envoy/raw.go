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

package envoy

import (
	"errors"

	envoy_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/golang/protobuf/ptypes/any"
	"google.golang.org/protobuf/proto"
	"sigs.k8s.io/yaml"

	util_proto "github.com/apache/dubbo-admin/pkg/common/util/proto"
)

func ResourceFromYaml(resYaml string) (proto.Message, error) {
	json, err := yaml.YAMLToJSON([]byte(resYaml))
	if err != nil {
		json = []byte(resYaml)
	}

	var anything any.Any
	if err := util_proto.FromJSON(json, &anything); err != nil {
		return nil, err
	}
	msg, err := anything.UnmarshalNew()
	if err != nil {
		return nil, err
	}
	p, ok := msg.(envoy_types.Resource)
	if !ok {
		return nil, errors.New("xDS resource doesn't implement all required interfaces")
	}
	if v, ok := p.(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return nil, err
		}
	}
	return p, nil
}
