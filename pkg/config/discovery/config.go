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

package discovery

import (
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/config"
)

type Type string

const (
	Zookeeper Type = "zookeeper"
	Nacos2    Type = "nacos2"
	// Mock is only for develop and test
	Mock Type = "mock"
)

const (
	defaultConfigWatchPeriod  = 30
	defaultServiceWatchPeriod = 30
)

// Config defines Discovery configuration
type Config struct {
	config.BaseConfig
	Name       string        `json:"name"`
	Type       Type          `json:"type"`
	Address    AddressConfig `json:"address"`
	Properties Properties    `json:"properties"`
}

// AddressConfig defines Discovery Engine address
type AddressConfig struct {
	Registry       string `json:"registry"`
	ConfigCenter   string `json:"configCenter"`
	MetadataReport string `json:"metadataReport"`
}
type Properties struct {
	ConfigWatchPeriod  int `json:"configWatchPeriod"`
	ServiceWatchPeriod int `json:"serviceWatchPeriod"`
}

func DefaultDiscoveryEnginConfig() *Config {
	return &Config{
		Name: "localhost",
		Type: Nacos2,
		Address: AddressConfig{
			Registry:       "nacos://127.0.0.1:8848?username=nacos&password=nacos",
			ConfigCenter:   "nacos://127.0.0.1:8848?username=nacos&password=nacos",
			MetadataReport: "nacos://127.0.0.1:8848?username=nacos&password=nacos",
		},
	}
}

func (c *Config) PreProcess() error {
	if strutil.IsBlank(c.Address.Registry) && c.Type != Mock {
		return bizerror.New(bizerror.ConfigError, "registry address is needed")
	}
	// if config center or metadata report is not set, use registry address
	if strutil.IsBlank(c.Address.ConfigCenter) {
		c.Address.ConfigCenter = c.Address.Registry
	}
	if strutil.IsBlank(c.Address.MetadataReport) {
		c.Address.MetadataReport = c.Address.Registry
	}
	if c.Properties.ServiceWatchPeriod <= 0 {
		c.Properties.ServiceWatchPeriod = defaultServiceWatchPeriod
		c.Properties.ConfigWatchPeriod = defaultConfigWatchPeriod
	}
	return nil
}

func (c *Config) Validate() error {
	if strutil.IsBlank(c.Name) {
		return bizerror.New(bizerror.InvalidArgument, "discovery name is needed")
	}
	if strutil.IsBlank(string(c.Type)) {
		return bizerror.New(bizerror.InvalidArgument, "discovery type is needed")
	}
	return nil
}
