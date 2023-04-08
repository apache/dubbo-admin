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

package config

import (
	"errors"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
)

type GovernanceConfig interface {
	SetConfig(key string, value string) error
	GetConfig(key string) (string, error)
	DeleteConfig(key string) error
	SetConfigWithGroup(group string, key string, value string) error
	GetConfigWithGroup(group string, key string) (string, error)
	DeleteConfigWithGroup(group string, key string) error
	Register(url *common.URL) error
	UnRegister(url *common.URL) error
}

type GovernanceConfigImpl struct{}

func (g *GovernanceConfigImpl) SetConfig(key string, value string) error {
	return g.SetConfigWithGroup(Group, key, value)
}

func (g *GovernanceConfigImpl) GetConfig(key string) (string, error) {
	return g.GetConfigWithGroup(Group, key)
}

func (g *GovernanceConfigImpl) DeleteConfig(key string) error {
	return g.DeleteConfigWithGroup(Group, key)
}

func (g *GovernanceConfigImpl) SetConfigWithGroup(group string, key string, value string) error {
	if key == "" || value == "" {
		return errors.New("key or value is empty")
	}
	return ConfigCenter.PublishConfig(key, group, value)
}

func (g *GovernanceConfigImpl) GetConfigWithGroup(group string, key string) (string, error) {
	if key == "" {
		return "", errors.New("key is empty")
	}
	return ConfigCenter.GetProperties(key, config_center.WithGroup(group))
}

func (g *GovernanceConfigImpl) DeleteConfigWithGroup(group string, key string) error {
	if key == "" {
		return errors.New("key is empty")
	}
	return ConfigCenter.RemoveConfig(key, group)
}

func (g *GovernanceConfigImpl) Register(url *common.URL) error {
	if url.String() == "" {
		return errors.New("url is empty")
	}
	return RegistryCenter.Register(url)
}

func (g *GovernanceConfigImpl) UnRegister(url *common.URL) error {
	if url.String() == "" {
		return errors.New("url is empty")
	}
	return RegistryCenter.UnRegister(url)
}
