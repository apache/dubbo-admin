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

	perrors "github.com/pkg/errors"

	gozk "github.com/dubbogo/go-zookeeper/zk"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
)

const group = "dubbo"

type RuleExists struct {
	cause error
}

func (exist *RuleExists) Error() string {
	return exist.cause.Error()
}

type RuleNotFound struct {
	cause error
}

func (notFound *RuleNotFound) Error() string {
	return notFound.cause.Error()
}

type GovernanceConfig interface {
	SetConfig(key string, value string) error
	GetConfig(key string) (string, error)
	DeleteConfig(key string) error
	SetConfigWithGroup(group string, key string, value string) error
	GetConfigWithGroup(group string, key string) (string, error)
	DeleteConfigWithGroup(group string, key string) error
	Register(url *common.URL) error
	UnRegister(url *common.URL) error
	GetList(group string) (map[string]string, error)
}

var impls map[string]func(cc config_center.DynamicConfiguration) GovernanceConfig

func init() {
	impls = map[string]func(cc config_center.DynamicConfiguration) GovernanceConfig{
		"zookeeper": func(cc config_center.DynamicConfiguration) GovernanceConfig {
			gc := &GovernanceConfigImpl{configCenter: cc}
			return &ZkGovImpl{
				GovernanceConfig: gc,
				configCenter:     cc,
				group:            group,
			}
		},
		"nacos": func(cc config_center.DynamicConfiguration) GovernanceConfig {
			gc := &GovernanceConfigImpl{configCenter: cc}
			return &NacosGovImpl{
				GovernanceConfig: gc,
				configCenter:     cc,
				group:            group,
			}
		},
	}
}

func NewGovernanceConfig(cc config_center.DynamicConfiguration, p string) GovernanceConfig {
	return impls[p](cc)
}

type GovernanceConfigImpl struct {
	configCenter config_center.DynamicConfiguration
}

func (g *GovernanceConfigImpl) SetConfig(key string, value string) error {
	return g.SetConfigWithGroup(group, key, value)
}

func (g *GovernanceConfigImpl) GetConfig(key string) (string, error) {
	return g.GetConfigWithGroup(group, key)
}

func (g *GovernanceConfigImpl) DeleteConfig(key string) error {
	return g.DeleteConfigWithGroup(group, key)
}

func (g *GovernanceConfigImpl) SetConfigWithGroup(group string, key string, value string) error {
	if key == "" || value == "" {
		return errors.New("key or value is empty")
	}
	return g.configCenter.PublishConfig(key, group, value)
}

func (g *GovernanceConfigImpl) GetConfigWithGroup(group string, key string) (string, error) {
	if key == "" {
		return "", errors.New("key is empty")
	}
	return g.configCenter.GetRule(key, config_center.WithGroup(group))
}

func (g *GovernanceConfigImpl) DeleteConfigWithGroup(group string, key string) error {
	if key == "" {
		return errors.New("key is empty")
	}
	return g.configCenter.RemoveConfig(key, group)
}

// Register only works for MockService
func (g *GovernanceConfigImpl) Register(url *common.URL) error {
	if url.String() == "" {
		return errors.New("url is empty")
	}
	return RegistryCenter.Register(url)
}

// UnRegister only works for MockService
func (g *GovernanceConfigImpl) UnRegister(url *common.URL) error {
	if url.String() == "" {
		return errors.New("url is empty")
	}
	return RegistryCenter.UnRegister(url)
}

func (g *GovernanceConfigImpl) GetList(group string) (map[string]string, error) {
	keys, err := g.configCenter.GetConfigKeysByGroup(group)
	if err != nil {
		return nil, err
	}

	list := make(map[string]string)
	for k := range keys.Items {
		rule, err := g.configCenter.GetRule(k.(string), config_center.WithGroup(group))
		if err != nil {
			return nil, err
		}
		list[k.(string)] = rule
	}
	return list, nil
}

type ZkGovImpl struct {
	GovernanceConfig
	configCenter config_center.DynamicConfiguration
	group        string
}

// GetConfig transform ZK specified 'node does not exist' err into unified admin rule error
func (zk *ZkGovImpl) GetConfig(key string) (string, error) {
	if key == "" {
		return "", errors.New("key is empty")
	}
	rule, err := zk.configCenter.GetRule(key, config_center.WithGroup(zk.group))
	if err != nil {
		if perrors.Is(err, gozk.ErrNoNode) {
			return "", &RuleNotFound{err}
		}
		return "", err
	}
	return rule, nil
}

// SetConfig transform ZK specified 'node already exist' err into unified admin rule error
func (zk *ZkGovImpl) SetConfig(key string, value string) error {
	if key == "" || value == "" {
		return errors.New("key or value is empty")
	}
	err := zk.configCenter.PublishConfig(key, zk.group, value)
	if err != nil {
		if perrors.Is(err, gozk.ErrNodeExists) {
			return &RuleExists{err}
		}
		return err
	}
	return nil
}

func (zk *ZkGovImpl) GetList(group string) (map[string]string, error) {
	return zk.GovernanceConfig.GetList(group)
}

type NacosGovImpl struct {
	GovernanceConfig
	configCenter config_center.DynamicConfiguration
	group        string
}

// GetConfig transform Nacos specified 'node does not exist' err into unified admin rule error
func (n *NacosGovImpl) GetConfig(key string) (string, error) {
	return n.GovernanceConfig.GetConfig(key)
}

// SetConfig transform Nacos specified 'node already exist' err into unified admin rule error
func (n *NacosGovImpl) SetConfig(key string, value string) error {
	return n.GovernanceConfig.SetConfig(key, value)
}

func (n *NacosGovImpl) GetList(group string) (map[string]string, error) {
	return n.GovernanceConfig.GetList(group)
}
