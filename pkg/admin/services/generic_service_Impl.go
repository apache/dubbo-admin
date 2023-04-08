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
)

type GenericServiceImpl struct{}

func (p *GenericServiceImpl) NewRefConf(appName, iface, protocol string) dubboConfig.ReferenceConfig {

	fromUrl := common.NewURLWithOptions(common.WithParams(config.RegistryCenter.GetURL().GetParams()))

	registryConfig := dubboConfig.RegistryConfig{}
	registryConfig.Group = fromUrl.GetParam("group", "")

	address, _ := common.NewURL(fromUrl.Protocol + "://" + fromUrl.Location)
	if fromUrl.GetParam(constant.NAMESPACE_KEY, "") != "" {
		address.AddParam(constant.NAMESPACE_KEY, fromUrl.GetParam(constant.NAMESPACE_KEY, ""))
	}
	registryConfig.Address = address.String()

	//registryConfig := &config.RegistryConfig{
	//	Protocol: "zookeeper",
	//	Address:  "127.0.0.1:2181",
	//	Group:    fromUrl.Group(),
	//}

	refConf := dubboConfig.ReferenceConfig{
		InterfaceName: iface,
		Cluster:       "failover",
		RegistryIDs:   []string{"zk"},
		Protocol:      protocol,
		Generic:       "true",
	}

	var registerMap map[string]*dubboConfig.RegistryConfig
	registerMap["zk"] = &registryConfig

	rootConfig := dubboConfig.RootConfig{
		Registries: registerMap,
	}

	_ = refConf.Init(&rootConfig)
	refConf.GenericLoad(appName)

	return refConf
}
