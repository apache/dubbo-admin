// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	dubboConfig "dubbo.apache.org/dubbo-go/v3/config"
	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/util"

	dubboconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	dubboconfig "dubbo.apache.org/dubbo-go/v3/config"
)

type GenericServiceImpl struct{}

func (p *GenericServiceImpl) NewRefConf(appName, iface, protocol string) dubboConfig.ReferenceConfig {
	fromUrl := config.AdminRegistry.Delegate().GetURL().Clone()

	registryConfig := dubboConfig.RegistryConfig{}
	registryConfig.Group = fromUrl.GetParam("group", "")
	address, _ := common.NewURL(fromUrl.Protocol + "://" + fromUrl.Location)
	if fromUrl.GetParam(constant.NamespaceKey, "") != "" {
		address.AddParam(constant.NamespaceKey, fromUrl.GetParam(constant.NamespaceKey, ""))
	}
	registryConfig.Address = address.String()
	registryConfig.RegistryType = dubboconstant.RegistryTypeInterface

	refConf := dubboConfig.ReferenceConfig{
		InterfaceName: util.GetInterface(iface),
		Group:         util.GetGroup(iface),
		Version:       util.GetVersion(iface),
		Cluster:       "failover",
		RegistryIDs:   []string{"genericRegistry"},
		Protocol:      protocol,
		Generic:       "true",
	}

	rootConfig := dubboconfig.NewRootConfigBuilder().
		AddRegistry("genericRegistry", &registryConfig).
		Build()
	if err := dubboconfig.Load(dubboconfig.WithRootConfig(rootConfig)); err != nil {
		panic(err)
	}
	_ = refConf.Init(rootConfig)
	refConf.GenericLoad(appName)

	return refConf
}
