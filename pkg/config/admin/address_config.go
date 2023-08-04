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

package admin

import (
	"net/url"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

type AddressConfig struct {
	Address string `yaml:"address"`
	Url     *url.URL
}

func (c *AddressConfig) Sanitize() {}

func (c *AddressConfig) Validate() error {
	return nil
}

func (c *AddressConfig) GetProtocol() string {
	return c.Url.Scheme
}

func (c *AddressConfig) GetAddress() string {
	return c.Url.Host
}

func (c *AddressConfig) GetUrlMap() url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.ConfigNamespaceKey, c.param("namespace", ""))
	urlMap.Set(constant.ConfigGroupKey, c.param(constant.GroupKey, "dubbo"))
	urlMap.Set(constant.MetadataReportGroupKey, c.param(constant.GroupKey, "dubbo"))
	urlMap.Set(constant.ClientNameKey, clientNameID(c.Url.Scheme, c.Url.Host))
	return urlMap
}

func (c *AddressConfig) param(key string, defaultValue string) string {
	param := c.Url.Query().Get(key)
	if len(param) > 0 {
		return param
	}
	return defaultValue
}

func (c *AddressConfig) ToURL() (*common.URL, error) {
	return common.NewURL(c.GetAddress(),
		common.WithProtocol(c.GetProtocol()),
		common.WithParams(c.GetUrlMap()),
		common.WithParamsValue("registry", c.GetProtocol()),
		common.WithUsername(c.param("username", "")),
		common.WithPassword(c.param("password", "")),
	)
}

func clientNameID(protocol, address string) string {
	return strings.Join([]string{protocol, address}, "-")
}
