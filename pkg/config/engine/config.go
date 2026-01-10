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

package engine

import (
	set "github.com/duke-git/lancet/v2/datastructure/set"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/config"
)

type Type string

const (
	VM         Type = "vm"
	Kubernetes Type = "kubernetes"
	Mock       Type = "mock"
)

type Config struct {
	config.BaseConfig
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Type       Type       `json:"type"`
	Properties Properties `json:"properties"`
}

func (c *Config) Validate() error {
	if strutil.IsBlank(c.ID) {
		return bizerror.New(bizerror.ConfigError, "engine id can not be empty")
	}
	return nil
}

type Properties struct {
	KubeConfigPath              string                       `json:"kubeConfigPath"`
	PodWatchSelector            string                       `json:"podWatchSelector"`
	DubboAppIdentifier          *KubernetesIdentifier        `json:"dubboAppIdentifier"`
	DubboRPCPortIdentifier      *KubernetesIdentifier        `json:"dubboRPCPortIdentifier"`
	DubboRegistryIdentifier     *KubernetesIdentifier        `json:"dubboRegistryIdentifier"`
	MainContainerChooseStrategy *MainContainerChooseStrategy `json:"mainContainerChooseStrategy"`
}

func (p *Properties) GetOrDefaultMainContainerChooseStrategy() *MainContainerChooseStrategy {
	if p.MainContainerChooseStrategy == nil ||
		!mainContainerChooseStrategyTypes.Contain(p.MainContainerChooseStrategy.Type) {
		return &MainContainerChooseStrategy{
			Type:  ChooseByIndex,
			Index: 0,
		}
	}
	return p.MainContainerChooseStrategy
}

type MainContainerChooseStrategyType string

const (
	ChooseByLast       MainContainerChooseStrategyType = "ByLast"
	ChooseByIndex      MainContainerChooseStrategyType = "ByIndex"
	ChooseByName       MainContainerChooseStrategyType = "ByName"
	ChooseByAnnotation MainContainerChooseStrategyType = "ByAnnotation"
)

var mainContainerChooseStrategyTypes = set.New(ChooseByLast, ChooseByIndex, ChooseByName, ChooseByAnnotation)

type MainContainerChooseStrategy struct {
	Type          MainContainerChooseStrategyType `json:"type"`
	Index         int                             `json:"index"`
	Name          string                          `json:"name"`
	AnnotationKey string                          `json:"annotationKey"`
}

type IdentifierType string

const (
	IdentifyByLabel      IdentifierType = "ByLabel"
	IdentifyByAnnotation IdentifierType = "ByAnnotation"
)

type KubernetesIdentifier struct {
	Type          IdentifierType `json:"type"`
	LabelKey      string         `json:"labelKey"`
	AnnotationKey string         `json:"annotationKey"`
}

func DefaultResourceEngineConfig() *Config {
	return &Config{
		Name:       "default",
		Type:       Mock,
		Properties: Properties{},
	}
}
