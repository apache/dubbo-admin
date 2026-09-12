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
	"reflect"
	"time"

	nacosconfigclient "github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	nacosnamingclient "github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	nacosvo "github.com/nacos-group/nacos-sdk-go/v2/vo"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/yaml"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/clients"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

type RuleGovernor struct {
	cfg          *discoverycfg.Config
	storeRouter  store.Router
	emitter      events.Emitter
	configClient nacosconfigclient.IConfigClient
	namingClient nacosnamingclient.INamingClient
}

func NewNacos2Governor(
	cfg *discoverycfg.Config,
	storeRouter store.Router,
	emitter events.Emitter) (*RuleGovernor, error) {
	configClient, namingClient, err := clients.CreateNacosClients(cfg.Address.ConfigCenter)
	if err != nil {
		return nil, err
	}
	return &RuleGovernor{
		cfg:          cfg,
		storeRouter:  storeRouter,
		emitter:      emitter,
		configClient: configClient,
		namingClient: namingClient,
	}, nil
}

func (g *RuleGovernor) CreateRule(r coremodel.Resource) error {
	rawContent, err := marshalRule(r)
	if err != nil {
		return bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("failed to marshal resource spec, res: %s", r.String()))
	}
	ok, err := g.configClient.PublishConfig(nacosvo.ConfigParam{
		DataId:  r.ResourceMeta().Name,
		Group:   constants.NacosConfigGroup,
		Content: string(rawContent),
	})
	if err != nil || !ok {
		logger.Errorf("failed to publish config in %s, res: %s", r.String(), r.ResourceMesh())
		return bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("failed to publish config, res: %s", r.String()))
	}
	// wait for the config to be published indeed
	<-time.After(2 * time.Second)
	g.GetConfigAndUpdateStore(r)
	return nil
}

func (g *RuleGovernor) UpdateRule(r coremodel.Resource) error {
	return g.CreateRule(r)
}

func (g *RuleGovernor) DeleteRule(r coremodel.Resource) error {
	ok, err := g.configClient.DeleteConfig(nacosvo.ConfigParam{
		DataId: r.ResourceMeta().Name,
		Group:  constants.NacosConfigGroup,
	})
	if err != nil || !ok {
		logger.Errorf("failed to delete config in %s, res: %s", r.String(), r.ResourceMesh())
		return bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("failed to delete config, res: %s", r.String()))
	}
	// delete resource in store, if delete failed, just log an error message
	// the lister-watcher will sync the deleted event finally
	st, err := g.storeRouter.ResourceKindRoute(r.ResourceKind())
	if err != nil {
		logger.Errorf("failed to get store in %s, res: %s, cause: %s", r.String(), r.ResourceMesh(), err)
		return nil
	}
	// wait for the config to be deleted indeed
	<-time.After(2 * time.Second)
	if err := st.Delete(r); err != nil {
		logger.Errorf("failed to delete resource in %s, res: %s, cause: %s", r.String(), r.ResourceMesh(), err)
		return nil
	}
	return nil
}

// GetConfigAndUpdateStore get resource from nacos, and update resource in store, if failed, just log an error message,
// the lister-watcher will sync the event finally
func (g *RuleGovernor) GetConfigAndUpdateStore(r coremodel.Resource) {
	content, err := g.configClient.GetConfig(nacosvo.ConfigParam{
		DataId: r.ResourceMeta().Name,
		Group:  constants.NacosConfigGroup,
	})
	if err != nil {
		logger.Errorf("failed to get config in %s, res: %s, cause: %s", r.String(), r.ResourceMesh(), err)
		return
	}
	st, err := g.storeRouter.ResourceKindRoute(r.ResourceKind())
	if err != nil {
		logger.Errorf("failed to get store in %s, res: %s, cause: %s", r.String(), r.ResourceMesh(), err)
		return
	}
	var res coremodel.Resource
	switch r.ResourceKind() {
	case meshresource.DynamicConfigKind:
		res = meshresource.ToDynamicConfigResource(r.ResourceMesh(), r.ResourceMeta().Name, content)
	case meshresource.ConditionRouteKind:
		res = meshresource.ToConditionRouteResource(r.ResourceMesh(), r.ResourceMeta().Name, content)
	case meshresource.TagRouteKind:
		res = meshresource.ToTagRouteResource(r.ResourceMesh(), r.ResourceMeta().Name, content)
	case meshresource.AffinityRouteKind:
		res = meshresource.ToAffinityRouteResource(r.ResourceMesh(), r.ResourceMeta().Name, content)
	case meshresource.ScriptRouteKind:
		res = meshresource.ToScriptRouteResource(r.ResourceMesh(), r.ResourceMeta().Name, content)
	}
	if res == nil {
		logger.Errorf("failed to decode config in %s, res: %s", r.String(), r.ResourceMesh())
		return
	}
	obj, exists, err := st.GetByKey(r.ResourceKey())
	if err != nil {
		logger.Errorf("failed to get resource in %s, res: %s, cause: %s", r.String(), r.ResourceMesh(), err)
		return
	}
	// if exists in store, update it
	if exists {
		if err := st.Update(res); err != nil {
			logger.Errorf("failed to update resource in %s, res: %s, cause: %s", r.String(), r.ResourceMesh(), err)
			return
		}
		oldRes, ok := obj.(coremodel.Resource)
		if !ok {
			logger.Errorf("type assertion failed in nacos2 discovery %s, expected Resource, got %s",
				g.mesh(), reflect.TypeOf(obj).Name())
		}
		g.emitter.Send(events.NewResourceChangedEvent(cache.Updated, oldRes, res))
		return
	}

	// otherwise add it
	err = st.Add(res)
	if err != nil {
		logger.Errorf("failed to add resource in %s, res: %s, cause: %s", r.String(), r.ResourceMesh(), err)
		return
	}
	g.emitter.Send(events.NewResourceChangedEvent(cache.Added, nil, res))
}

func marshalRule(r coremodel.Resource) ([]byte, error) {
	if r.ResourceKind() == meshresource.AffinityRouteKind || r.ResourceKind() == meshresource.ScriptRouteKind {
		return meshresource.EncodeRule(r)
	}
	return yaml.Marshal(r.ResourceSpec())
}

func (g *RuleGovernor) mesh() string {
	return g.cfg.ID
}
