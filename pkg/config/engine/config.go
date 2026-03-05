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
	"fmt"

	set "github.com/duke-git/lancet/v2/datastructure/set"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/google/uuid"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/config"
)

type Type string

var supportedEngineTypes = set.New(Kubernetes, Mock)

const (
	Kubernetes Type = "kubernetes"
	Mock       Type = "mock"
)

type Config struct {
	config.BaseConfig
	ID         string     `json:"id" yaml:"id"`
	Name       string     `json:"name" yaml:"name"`
	Type       Type       `json:"type" yaml:"type"`
	Properties Properties `json:"properties" yaml:"properties"`
}

func (c *Config) Validate() error {
	if strutil.IsBlank(c.ID) {
		return bizerror.New(bizerror.ConfigError, "engine id can not be empty")
	}
	if !supportedEngineTypes.Contain(c.Type) {
		return bizerror.New(bizerror.ConfigError, fmt.Sprintf("engine type %s is not supported", c.Type))
	}
	return c.Properties.Validate()
}

type Properties struct {
	config.BaseConfig
	KubeConfigPath              string                       `json:"kubeConfigPath" yaml:"kubeConfigPath"`
	PodWatchSelector            string                       `json:"podWatchSelector" yaml:"podWatchSelector"`
	DubboAppIdentifier          *KubernetesIdentifier        `json:"dubboAppIdentifier" yaml:"dubboAppIdentifier"`
	DubboRPCPortIdentifier      *KubernetesIdentifier        `json:"dubboRPCPortIdentifier" yaml:"dubboRPCPortIdentifier"`
	DubboDiscoveryIdentifier    *KubernetesIdentifier        `json:"dubboDiscoveryIdentifier" yaml:"dubboDiscoveryIdentifier"`
	MainContainerChooseStrategy *MainContainerChooseStrategy `json:"mainContainerChooseStrategy" yaml:"mainContainerChooseStrategy"`
}

func (p *Properties) Validate() error {
	if p.DubboAppIdentifier != nil {
		if err := p.DubboAppIdentifier.Validate(); err != nil {
			return err
		}
	}
	if p.DubboRPCPortIdentifier != nil {
		if err := p.DubboRPCPortIdentifier.Validate(); err != nil {
			return err
		}
	}
	if p.DubboDiscoveryIdentifier != nil {
		if err := p.DubboDiscoveryIdentifier.Validate(); err != nil {
			return err
		}
	}
	if p.MainContainerChooseStrategy != nil {
		if err := p.MainContainerChooseStrategy.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type MainContainerChooseStrategyType string

var mainContainerChooseStrategyTypes = set.New(ChooseByLast, ChooseByIndex, ChooseByName, ChooseByAnnotation)

const (
	ChooseByLast       MainContainerChooseStrategyType = "ByLast"
	ChooseByIndex      MainContainerChooseStrategyType = "ByIndex"
	ChooseByName       MainContainerChooseStrategyType = "ByName"
	ChooseByAnnotation MainContainerChooseStrategyType = "ByAnnotation"
)

type MainContainerChooseStrategy struct {
	config.BaseConfig
	Type          MainContainerChooseStrategyType `json:"type"`
	Index         int                             `json:"index"`
	Name          string                          `json:"name"`
	AnnotationKey string                          `json:"annotationKey"`
}

func (m *MainContainerChooseStrategy) Validate() error {
	if !mainContainerChooseStrategyTypes.Contain(m.Type) {
		return bizerror.New(bizerror.ConfigError, fmt.Sprintf("unsupported main container choose strategy type: %s", m.Type))
	}
	return nil
}

type IdentifierType string

var identifierTypes = set.New(IdentifyByLabel, IdentifyByAnnotation)

const (
	IdentifyByLabel      IdentifierType = "ByLabel"
	IdentifyByAnnotation IdentifierType = "ByAnnotation"
)

type KubernetesIdentifier struct {
	config.BaseConfig
	Type          IdentifierType `json:"type"`
	LabelKey      string         `json:"labelKey"`
	AnnotationKey string         `json:"annotationKey"`
}

func (k *KubernetesIdentifier) Validate() error {
	if !identifierTypes.Contain(k.Type) {
		return bizerror.New(bizerror.ConfigError, fmt.Sprintf("unsupported identifier type: %s", k.Type))
	}
	return nil
}

func DefaultResourceEngineConfig() *Config {
	return &Config{
		ID:   uuid.New().String(),
		Name: "default",
		Type: Mock,
		Properties: Properties{
			MainContainerChooseStrategy: &MainContainerChooseStrategy{
				Type:  ChooseByIndex,
				Index: 0,
			},
		},
	}
}
