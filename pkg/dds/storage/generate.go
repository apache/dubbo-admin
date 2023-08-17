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
	"github.com/apache/dubbo-admin/pkg/core/tools/generate"
	"google.golang.org/protobuf/types/known/anypb"
)

type DdsResourceGenerator interface {
	Generate(data []model.Config, endpoint *endpoint.Endpoint) ([]*anypb.Any, error)
}

type AuthenticationGenerator struct{}

func (g *AuthenticationGenerator) Generate(data []model.Config, endpoint *endpoint.Endpoint) ([]*anypb.Any, error) {
	res := make([]*anypb.Any, 0)
	for _, v := range data {
		policy := v.Spec.(*api.AuthenticationPolicy)
		toClient := &api.AuthenticationPolicyToClient{
			Spec: &api.AuthenticationSpecToClient{},
		}
		key := generate.GenerateKey(v.Name, v.Namespace)
		toClient.Key = key
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
		toClient.Spec.Action = policy.Action
		if policy.GetPortLevel() != nil {
			toClient.Spec.PortLevel = make([]*api.AuthenticationPolicyPortLevel, 0, len(policy.PortLevel))
			for _, portLevel := range policy.PortLevel {
				toClient.Spec.PortLevel = append(toClient.Spec.PortLevel, &api.AuthenticationPolicyPortLevel{
					Port:   portLevel.Port,
					Action: portLevel.Action,
				})
			}
		}

		gogo, err := model.ToProtoGogo(toClient)
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
		policy := v.Spec.(*api.AuthorizationPolicy)
		toClient := &api.AuthorizationPolicyToClient{}
		key := generate.GenerateKey(v.Name, v.Namespace)
		toClient.Key = key
		if policy.GetRules() != nil {
			match := true
			for _, policyRule := range policy.Rules {
				if policyRule.GetTo() == nil {
					policyRule.To = &api.AuthorizationPolicyTarget{}
				}
				if !MatchAuthrSelector(policyRule.To, endpoint) {
					match = false
					break
				}
			}
			if !match {
				continue
			}

			toClient.Spec = &api.AuthorizationPolicySpecToClient{}

			toClient.Spec.Action = policy.Action
			toClient.Spec.Samples = policy.Samples
			toClient.Spec.Order = policy.Order
			toClient.Spec.MatchType = policy.MatchType

			if policy.Rules != nil {
				toClient.Spec.Rules = make([]*api.AuthorizationPolicyRuleToClient, 0, len(policy.Rules))
				for _, rule := range policy.Rules {
					if rule.GetFrom() == nil {
						rule.From = &api.AuthorizationPolicySource{}
					}
					if rule.GetWhen() == nil {
						rule.When = &api.AuthorizationPolicyCondition{}
					}
					ruleToClient := &api.AuthorizationPolicyRuleToClient{
						From: rule.From.DeepCopy(),
						When: rule.When.DeepCopy(),
					}
					toClient.Spec.Rules = append(toClient.Spec.Rules, ruleToClient)
				}
			}
		}
		gogo, err := model.ToProtoGogo(toClient)
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
		toClient := &api.ConditionRouteToClient{}
		key := generate.GenerateKey(config.Name, config.Namespace)
		toClient.Key = key
		toClient.Spec = config.Spec.(*api.ConditionRoute)
		gogo, err := model.ToProtoGogo(toClient)
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
		toClient := &api.DynamicConfigToClient{}
		key := generate.GenerateKey(config.Name, config.Namespace)
		toClient.Key = key
		toClient.Spec = config.Spec.(*api.DynamicConfig)
		gogo, err := model.ToProtoGogo(toClient)
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
		toClient := &api.ServiceNameMappingToClient{}
		key := generate.GenerateKey(config.Name, config.Namespace)
		toClient.Key = key
		toClient.Spec = config.Spec.(*api.ServiceNameMapping)
		gogo, err := model.ToProtoGogo(toClient)
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
		toClient := &api.TagRouteToClient{}
		key := generate.GenerateKey(config.Name, config.Namespace)
		toClient.Key = key
		toClient.Spec = config.Spec.(*api.TagRoute)
		gogo, err := model.ToProtoGogo(toClient)
		if err != nil {
			return nil, err
		}
		res = append(res, gogo)
	}
	return res, nil
}
