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
	"net/url"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

type AddressConfig struct {
	Address string `yaml:"address"`
	url     *url.URL
}

func (c *AddressConfig) getProtocol() string {
	return c.url.Scheme
}

func (c *AddressConfig) getAddress() string {
	return c.url.Host
}

func (c *AddressConfig) GetUrlMap() url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.ConfigNamespaceKey, c.param("namespace", ""))
	urlMap.Set(constant.ConfigGroupKey, c.param("group", ""))
	return urlMap
}

func (c *AddressConfig) param(key string, defaultValue string) string {
	param := c.url.Query().Get(key)
	if len(param) > 0 {
		return param
	}
	return defaultValue
}

func (c *AddressConfig) toURL() (*common.URL, error) {
	return common.NewURL(c.getAddress(),
		common.WithProtocol(c.getProtocol()),
		common.WithParams(c.GetUrlMap()),
		common.WithUsername(c.param("username", "")),
		common.WithPassword(c.param("password", "")),
	)
}
