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

package nacos2

import (
	"fmt"

	dubbogocom "dubbo.apache.org/dubbo-go/v3/common"
	dubbogoconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	dubbogonacos "dubbo.apache.org/dubbo-go/v3/remoting/nacos"
	nacosconfigclient "github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	nacosnamingclient "github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/discovery"
	"github.com/apache/dubbo-admin/pkg/core/events"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/discovery/nacos2/listerwatcher"
)

func init() {
	discovery.RegisterListWatcherFactory(&Factory{
		subscribers: make([]events.Subscriber, 0),
	})
}

type Factory struct {
	subscribers []events.Subscriber
}

func (f *Factory) Support(d discoverycfg.Type) bool {
	return d == discoverycfg.Nacos2
}

func (f *Factory) NewListWatchers(
	cfg *discoverycfg.Config) ([]controller.ResourceListerWatcher, error) {
	nacosConfigClient, nacosNamingClient, err := f.initNacosClients(cfg)
	listerWatchers, err := f.initListerWatchers(cfg, nacosConfigClient, nacosNamingClient)
	if err != nil {
		return nil, err
	}
	return listerWatchers, nil
}

func (f *Factory) initNacosClients(
	cfg *discoverycfg.Config,
) (nacosconfigclient.IConfigClient, nacosnamingclient.INamingClient, error) {
	cfgCenterUrl, err := dubbogocom.NewURL(cfg.Address.ConfigCenter)
	if err != nil {
		return nil, nil, err
	}
	cfgCenterUrl.AddParam(dubbogoconstant.ClientNameKey, cfg.ID)
	nacosConfigClient, err := dubbogonacos.NewNacosConfigClientByUrl(cfgCenterUrl)
	if err != nil {
		return nil, nil, bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("cannot create nacos config client for %s %s", cfg.ID, cfg.Address))
	}
	registryUrl, err := dubbogocom.NewURL(cfg.Address.Registry)
	if err != nil {
		return nil, nil, err
	}
	registryUrl.AddParam(dubbogoconstant.ClientNameKey, cfg.ID)
	namingClient, err := dubbogonacos.NewNacosClientByURL(registryUrl)
	if err != nil {
		return nil, nil, bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("cannot create nacos naming client for %s %s", cfg.ID, cfg.Address))
	}
	return nacosConfigClient.Client(), namingClient.Client(), nil
}

func (f *Factory) initListerWatchers(
	cfg *discoverycfg.Config,
	nacosConfigClient nacosconfigclient.IConfigClient,
	namingClient nacosnamingclient.INamingClient) ([]controller.ResourceListerWatcher, error) {
	nacosServiceLW := listerwatcher.NewNacosServiceListerWatcher(cfg, namingClient)
	dynamicConfigLW, err := listerwatcher.NewConfigListerWatcher(
		meshresource.DynamicConfigKind,
		cfg,
		nacosConfigClient,
		meshresource.ToDynamicConfigResource,
		true,
		constants.WildcardCharacter+constants.ConfiguratorRuleDotSuffix,
		constants.NacosConfigGroup,
	)
	if err != nil {
		return nil, err
	}
	conditionRouteLW, err := listerwatcher.NewConfigListerWatcher(
		meshresource.ConditionRouteKind,
		cfg,
		nacosConfigClient,
		meshresource.ToConditionRouteResource,
		true,
		constants.WildcardCharacter+constants.ConditionRuleDotSuffix,
		constants.NacosConfigGroup,
	)
	if err != nil {
		return nil, err
	}
	tagRouteLW, err := listerwatcher.NewConfigListerWatcher(
		meshresource.TagRouteKind,
		cfg,
		nacosConfigClient,
		meshresource.ToTagRouteResource,
		true,
		constants.WildcardCharacter+constants.TagRuleDotSuffix, // "*.tag-router"
		constants.NacosConfigGroup,
	)
	if err != nil {
		return nil, err
	}
	affinityRouteLW, err := listerwatcher.NewConfigListerWatcher(
		meshresource.AffinityRouteKind,
		cfg,
		nacosConfigClient,
		meshresource.ToAffinityRouteResource,
		true,
		constants.WildcardCharacter+constants.AffinityRuleDotSuffix,
		constants.NacosConfigGroup,
	)
	if err != nil {
		return nil, err
	}
	scriptRouteLW, err := listerwatcher.NewConfigListerWatcher(
		meshresource.ScriptRouteKind,
		cfg,
		nacosConfigClient,
		meshresource.ToScriptRouteResource,
		true,
		constants.WildcardCharacter+constants.ScriptRuleDotSuffix,
		constants.NacosConfigGroup,
	)
	if err != nil {
		return nil, err
	}
	serviceProviderMetadataLW, err := listerwatcher.NewConfigListerWatcher(
		meshresource.ServiceProviderMetadataKind,
		cfg,
		nacosConfigClient,
		meshresource.ToServiceProviderMetadataResource,
		true,
		constants.ServiceProviderNacosKey,
		constants.NacosConfigGroup,
	)
	if err != nil {
		return nil, err
	}
	serviceProviderMappingLW, err := listerwatcher.NewConfigListerWatcher(
		meshresource.ServiceProviderMappingKind,
		cfg,
		nacosConfigClient,
		meshresource.ToServiceProviderMappingResource,
		false,
		"",
		constants.NacosMappingGroup,
	)
	if err != nil {
		return nil, err
	}
	return []controller.ResourceListerWatcher{
		nacosServiceLW,
		dynamicConfigLW,
		conditionRouteLW,
		tagRouteLW,
		affinityRouteLW,
		scriptRouteLW,
		serviceProviderMetadataLW,
		serviceProviderMappingLW,
	}, nil
}
