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

	"github.com/dubbogo/gost/encoding/yaml"

	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
)

// Application 流量管控相关的基础label
const (
	ApplicationLabel = "dubbo.io/application"
	ServiceLabel        = "dubbo.io/service"
	IDLabel             = "dubbo.io/id"
	ServiceVersionLabel = "dubbo.io/serviceVersion"
	ServiceGroupLabel = "dubbo.io/serviceGroup"
	RevisionLabel     = "dubbo.io/revision"
)

type Base struct {
	Application    string `json:"application" yaml:"application"`
	Service        string `json:"service" yaml:"service"`
	ID             string `json:"id" yaml:"id"`
	ServiceVersion string `json:"serviceVersion" yaml:"serviceVersion"`
	ServiceGroup   string `json:"serviceGroup" yaml:"serviceGroup"`
}

func BuildServiceKey(baseDto Base) string {
	if baseDto.Application != "" {
		return baseDto.Application
	}
	// id format: "${class}:${version}:${group}"
	return baseDto.Service + consts.Colon + baseDto.ServiceVersion + consts.Colon + baseDto.ServiceGroup
}

func GetRoutePath(key string, routeType string) string {
	key = strings.ReplaceAll(key, "/", "*")
	switch routeType {
	case consts.ConditionRoute:
		return key + consts.ConditionRuleSuffix
	case consts.TagRoute:
		return key + consts.TagRuleSuffix
	case consts.AffinityRoute:
		return key + consts.AffinityRuleSuffix
	}
	return key + "." + routeType + "-router"
}

func LoadObject(content string, obj interface{}) error {
	return yaml.UnmarshalYML([]byte(content), obj)
}
