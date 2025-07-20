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

package v1alpha1

import (
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	"github.com/apache/dubbo-admin/pkg/common/util/proto"
	"github.com/apache/dubbo-admin/pkg/core/consts"
)

func (x *ConditionRoute) GetVersion() string {
	if x.ToConditionRouteV3() != nil {
		return consts.ConfiguratorVersionV3
	} else {
		return consts.ConfiguratorVersionV3x1
	}
}

func (x *ConditionRoute) ToYAML() ([]byte, error) {
	if msg := x.ToConditionRouteV3x1(); msg != nil {
		return proto.ToYAML(msg)
	} else if msg := x.ToConditionRouteV3(); msg != nil {
		return proto.ToYAML(msg)
	}
	return nil, errors.New(`ConditionRoute validation failed`)
}

func ConditionRouteDecodeFromYAML(content []byte) (*ConditionRoute, error) {
	_map := map[string]interface{}{}
	err := yaml.Unmarshal(content, &_map)
	if err != nil {
		return nil, err
	}

	version, ok := _map[consts.ConfigVersionKey].(string)
	if !ok {
		return nil, errors.New("invalid condition route format")
	}
	if version == consts.ConfiguratorVersionV3 {
		v3 := new(ConditionRouteV3)
		if err = proto.FromYAML(content, v3); err != nil {
			return nil, err
		}
		return v3.ToConditionRoute(), nil
	} else if version == consts.ConfiguratorVersionV3x1 {
		v3x1 := new(ConditionRouteV3X1)
		if err = proto.FromYAML(content, v3x1); err != nil {
			return nil, err
		}
		return v3x1.ToConditionRoute(), nil
	} else {
		return nil, errors.New("invalid condition route format")
	}
}

func (x *ConditionRouteV3) ToConditionRoute() *ConditionRoute {
	return &ConditionRoute{Conditions: &ConditionRoute_ConditionsV3{ConditionsV3: x}}
}

func (x *ConditionRouteV3X1) ToConditionRoute() *ConditionRoute {
	return &ConditionRoute{Conditions: &ConditionRoute_ConditionsV3X1{ConditionsV3X1: x}}
}

func (x *ConditionRoute) ToConditionRouteV3() *ConditionRouteV3 {
	if v, ok := x.Conditions.(*ConditionRoute_ConditionsV3); ok {
		return v.ConditionsV3
	}
	return nil
}

func (x *ConditionRoute) ToConditionRouteV3x1() *ConditionRouteV3X1 {
	if v, ok := x.Conditions.(*ConditionRoute_ConditionsV3X1); ok {
		return v.ConditionsV3X1
	}
	return nil
}

func (x *ConditionRouteV3X1) RangeConditions(f func(r *ConditionRule) (isStop bool)) {
	if f == nil {
		return
	}
	for _, condition := range x.Conditions {
		if condition != nil && f(condition) {
			break
		}
	}
}

func (x *ConditionRouteV3) RangeConditions(f func(condition string) (isStop bool)) {
	if f == nil {
		return
	}
	for _, condition := range x.Conditions {
		if f(condition) {
			break
		}
	}
}

func (x *ConditionRouteV3X1) RangeConditionsToRemove(f func(r *ConditionRule) (isRemove bool)) {
	if f == nil {
		return
	}
	res := make([]*ConditionRule, 0, len(x.Conditions)/2+1)
	for _, condition := range x.Conditions {
		if condition != nil && !f(condition) {
			res = append(res, condition)
		}
	}
	x.Conditions = res
}

func (x *ConditionRule) IsMatchMethod() (string, bool) {
	conditions := strings.Split(x.From.Match, "&")
	for _, condition := range conditions {
		if idx := strings.Index(condition, "method"); idx != -1 {
			args := strings.Split(condition, consts.Equal)
			if len(args) == 2 {
				return strings.TrimSpace(args[1]), true
			}
		}
	}
	return "", false
}
