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

import "github.com/apache/dubbo-admin/pkg/config"

type Type string

const (
	zookeeper Type = "zookeeper"
	nacos     Type = "nacos"
)

// Config defines Discovery configuration
type Config struct {
	config.BaseConfig
	Name    string `json:"name"`
	Type    Type   `json:"type"`
	Address AddressConfig
}

// AddressConfig defines Discovery Engine address
type AddressConfig struct {
	Registry       string `json:"registry"`
	ConfigCenter   string `json:"configCenter"`
	MetadataReport string `json:"metadataReport"`
}

func DefaultDiscoveryEnginConfig() *Config {
	return &Config{
		Name: "localhost",
		Type: nacos,
		Address: AddressConfig{
			Registry:       "nacos://127.0.0.1:8848?username=nacos&password=nacos",
			ConfigCenter:   "nacos://127.0.0.1:8848?username=nacos&password=nacos",
			MetadataReport: "nacos://127.0.0.1:8848?username=nacos&password=nacos",
		},
	}
}
