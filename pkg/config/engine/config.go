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

import "github.com/apache/dubbo-admin/pkg/config"

type Type string

const (
	VM         Type = "vm"
	Kubernetes Type = "kubernetes"
)

type Config struct {
	config.BaseConfig
	Name       string     `json:"name"`
	Type       Type       `json:"type"`
	Properties Properties `json:"properties"`
}

type Properties struct {
	KubeConfigPath              string                       `json:"kubeConfigPath"`
	PodWatchSelector            string                       `json:"podWatchSelector"`
	DubboAppIdentifier          *DubboAppIdentifier          `json:"dubboAppIdentifier"`
	MainContainerChooseStrategy *MainContainerChooseStrategy `json:"mainContainerChooseStrategy"`
}

func (p *Properties) GetOrDefaultMainContainerChooseStrategy() *MainContainerChooseStrategy {
	if p.MainContainerChooseStrategy == nil {
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

type MainContainerChooseStrategy struct {
	Type          MainContainerChooseStrategyType `json:"type"`
	Index         int                             `json:"index"`
	Name          string                          `json:"name"`
	AnnotationKey string                          `json:"annotationKey"`
}

type DubboAppIdentifierType string

const (
	IdentifyByLabel      DubboAppIdentifierType = "ByLabel"
	IdentifyByAnnotation DubboAppIdentifierType = "ByAnnotation"
	IdentifyByIP         DubboAppIdentifierType = "ByIP"
)

type DubboAppIdentifier struct {
	Type          DubboAppIdentifierType `json:"type"`
	LabelKey      string                 `json:"labelKey"`
	AnnotationKey string                 `json:"annotationKey"`
}

func DefaultResourceEngineConfig() *Config {
	return &Config{
		Name:       "default",
		Type:       VM,
		Properties: Properties{},
	}
}
