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

package clients

import (
	"fmt"

	dubbogocom "dubbo.apache.org/dubbo-go/v3/common"
	dubbogoconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	dubbogonacos "dubbo.apache.org/dubbo-go/v3/remoting/nacos"
	nacosconfigclient "github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	nacosnamingclient "github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
)

// CreateNacosClients creates nacos clients by url
// format: nacos://host:port?username=${uname}&password=${pwd}
func CreateNacosClients(url string) (nacosconfigclient.IConfigClient, nacosnamingclient.INamingClient, error) {
	nacosUrl, err := dubbogocom.NewURL(url)
	if err != nil {
		return nil, nil, err
	}
	nacosUrl.AddParam(dubbogoconstant.ClientNameKey, url)
	configClient, err := dubbogonacos.NewNacosConfigClientByUrl(nacosUrl)
	if err != nil {
		return nil, nil, bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("cannot create nacos config client for %s", url))
	}
	namingClient, err := dubbogonacos.NewNacosClientByURL(nacosUrl)
	if err != nil {
		return nil, nil, bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("cannot create nacos naming client for %s", url))
	}
	return configClient.Client(), namingClient.Client(), nil
}
