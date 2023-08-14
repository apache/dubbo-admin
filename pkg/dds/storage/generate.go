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

package storage

import (
	api "github.com/apache/dubbo-admin/api/resource/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/model"
	"google.golang.org/protobuf/types/known/anypb"
)

type DdsResourceGenerator interface {
	Generate(data []model.Config, endpoint *endpoint.Endpoint) ([]*anypb.Any, error)
}

type AuthenticationGenerator struct{}

func (g *AuthenticationGenerator) Generate(data []model.Config, endpoint *endpoint.Endpoint) ([]*anypb.Any, error) {
	res := make([]*anypb.Any, 0)
	for _, v := range data {
		deepCopy := v.DeepCopy()
		policy := deepCopy.Spec.(*api.AuthenticationPolicy)
		if policy.GetSelector() != nil {
			match := true
			for _, selector := range policy.Selector {
				if !MatchAuthnSelector(selector, endpoint) {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}
		policy.Selector = nil
		gogo, err := model.ToProtoGogo(policy)
		if err != nil {
			return nil, err
		}
		res = append(res, gogo)
	}
	return res, nil
}

type AuthorizationGenerator struct{}

func (g *AuthorizationGenerator) Generate(data []model.Config, endpoint *endpoint.Endpoint) ([]*anypb.Any, error) {
	res := make([]*anypb.Any, 0)
	for _, v := range data {
		deepCopy := v.DeepCopy()
		policy := deepCopy.Spec.(*api.AuthorizationPolicy)
		if policy.GetRules() != nil {
			match := true
			for _, policyRule := range policy.Rules {
				if !MatchAuthrSelector(policyRule.To, endpoint) {
					match = false
					break
				}
				policyRule.To = nil
			}
			if !match {
				continue
			}
		}
		gogo, err := model.ToProtoGogo(policy)
		if err != nil {
			return nil, err
		}
		res = append(res, gogo)
	}
	return res, nil
}

type ConditionRoutesGenerator struct{}

func (g *ConditionRoutesGenerator) Generate(data []model.Config, endpoint *endpoint.Endpoint) ([]*anypb.Any, error) {
	res := make([]*anypb.Any, 0)
	for _, config := range data {
		gogo, err := model.ToProtoGogo(config.Spec.(*api.ConditionRoute))
		if err != nil {
			return nil, err
		}
		res = append(res, gogo)
	}
	return res, nil
}

type DynamicConfigsGenerator struct{}

func (g *DynamicConfigsGenerator) Generate(data []model.Config, endpoint *endpoint.Endpoint) ([]*anypb.Any, error) {
	res := make([]*anypb.Any, 0)
	for _, config := range data {
		gogo, err := model.ToProtoGogo(config.Spec.(*api.DynamicConfig))
		if err != nil {
			return nil, err
		}
		res = append(res, gogo)
	}
	return res, nil
}

type ServiceMappingGenerator struct{}

func (g *ServiceMappingGenerator) Generate(data []model.Config, endpoint *endpoint.Endpoint) ([]*anypb.Any, error) {
	res := make([]*anypb.Any, 0)
	for _, config := range data {
		gogo, err := model.ToProtoGogo(config.Spec.(*api.ServiceNameMapping))
		if err != nil {
			return nil, err
		}
		res = append(res, gogo)
	}
	return res, nil
}

type TagRoutesGenerator struct{}

func (g *TagRoutesGenerator) Generate(data []model.Config, endpoint *endpoint.Endpoint) ([]*anypb.Any, error) {
	res := make([]*anypb.Any, 0)
	for _, config := range data {
		gogo, err := model.ToProtoGogo(config.Spec.(*api.TagRoute))
		if err != nil {
			return nil, err
		}
		res = append(res, gogo)
	}
	return res, nil
}
