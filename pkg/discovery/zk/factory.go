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

package zk

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/discovery"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/discovery/zk/listerwatcher"
)

func init() {
	discovery.RegisterListWatcherFactory(&Factory{})
}

type Factory struct {
}

func (f *Factory) Support(typ discoverycfg.Type) bool {
	return discoverycfg.Zookeeper == typ
}

func (f *Factory) NewListWatchers(config *discoverycfg.Config) ([]controller.ResourceListerWatcher, error) {
	mappingLw, err := listerwatcher.NewListerWatcher(
		meshresource.ServiceProviderMappingKind,
		toUpsertMappingResource,
		toDeleteMappingResource,
		"/dubbo/mapping",
		config,
	)
	if err != nil {
		return nil, err
	}
	rpcInstanceLW, err := listerwatcher.NewListerWatcher(
		meshresource.RPCInstanceKind,
		toUpsertRPCInstanceResource,
		toDeleteRPCInstanceResource,
		"/services",
		config,
	)
	if err != nil {
		return nil, err
	}
	// Keep the config watcher connected to the configured config-center
	// endpoint. Registry and config-center addresses are allowed to differ.
	configLW, err := listerwatcher.NewListerWatcherWithAddress(
		meshresource.ZKConfigKind,
		toUpsertZKConfigResource,
		toDeleteZKConfigResource,
		"/dubbo/config",
		config,
		config.Address.ConfigCenter,
	)
	if err != nil {
		return nil, err
	}
	metadataLW, err := listerwatcher.NewListerWatcher(
		meshresource.ZKMetadataKind,
		toUpsertZKMetadataResource,
		toDeleteZKMetadataResource,
		"/dubbo/metadata",
		config,
	)
	if err != nil {
		return nil, err
	}

	return []controller.ResourceListerWatcher{
		mappingLw,
		rpcInstanceLW,
		configLW,
		metadataLW,
	}, nil
}

func toUpsertMappingResource(mesh, nodePath, nodeData string) coremodel.Resource {
	paths := strings.Split(nodePath, constants.PathSeparator)
	if len(paths) != 4 {
		return nil
	}
	serviceName := paths[3]
	return meshresource.ToServiceProviderMappingResource(mesh, serviceName, nodeData)
}

func toDeleteMappingResource(mesh, nodePath string) coremodel.Resource {
	paths := strings.Split(nodePath, constants.PathSeparator)
	if len(paths) != 4 {
		return nil
	}
	serviceName := paths[3]
	res := meshresource.NewServiceProviderMappingResourceWithAttributes(serviceName, mesh)
	return res
}

func toUpsertZKConfigResource(mesh, nodePath, nodeData string) coremodel.Resource {
	configName, ok := zkConfigName(nodePath)
	if !ok {
		return nil
	}
	res := meshresource.NewZKConfigResourceWithAttributes(configName, mesh)
	res.Spec = &meshproto.ZKConfig{
		NodeName: configName,
		NodeData: nodeData,
	}
	return res
}

func toDeleteZKConfigResource(mesh, nodePath string) coremodel.Resource {
	configName, ok := zkConfigName(nodePath)
	if !ok {
		return nil
	}
	res := meshresource.NewZKConfigResourceWithAttributes(configName, mesh)
	res.Spec = &meshproto.ZKConfig{
		NodeName: configName,
	}
	return res
}

// zkConfigName accepts the Dubbo 3 grouped config path
// /dubbo/config/dubbo/<ruleName> and the legacy flat path
// /dubbo/config/<ruleName>. Group/root nodes are not rules.
func zkConfigName(nodePath string) (string, bool) {
	paths := strings.Split(strings.Trim(nodePath, constants.PathSeparator), constants.PathSeparator)
	// Trim above removes the leading slash, therefore the grouped form has
	// [dubbo config dubbo rule] and the legacy form [dubbo config rule].
	switch len(paths) {
	case 3:
		if paths[2] != "" && paths[2] != constants.RuleConfigGroup {
			return paths[2], true
		}
	case 4:
		if paths[2] == constants.RuleConfigGroup && paths[3] != "" {
			return paths[3], true
		}
	}
	return "", false
}

func toUpsertZKMetadataResource(mesh, nodePath, nodeData string) coremodel.Resource {
	paths := strings.Split(nodePath, constants.PathSeparator)
	if len(paths) < 5 {
		return nil
	}
	res := meshresource.NewZKMetadataResourceWithAttributes(nodePath, mesh)
	res.Spec = &meshproto.ZKMetadata{
		NodePath: nodePath,
		NodeData: nodeData,
	}
	return res
}

func toDeleteZKMetadataResource(mesh, nodePath string) coremodel.Resource {
	paths := strings.Split(nodePath, constants.PathSeparator)
	if len(paths) < 5 {
		return nil
	}
	return meshresource.NewZKMetadataResourceWithAttributes(nodePath, mesh)
}

type ZooKeeperInstance struct {
	Name                string                    `json:"name"`
	ID                  string                    `json:"id"`
	Address             string                    `json:"address"`
	Port                int64                     `json:"port"`
	SSLPort             *int                      `json:"sslPort,omitempty"`
	Payload             *ZookeeperInstancePayload `json:"payload"`
	RegistrationTimeUTC int64                     `json:"registrationTimeUTC"`
	ServiceType         string                    `json:"serviceType"`
	UriSpec             *string                   `json:"uriSpec,omitempty"`
}

type ZookeeperInstancePayload struct {
	Class    string            `json:"@class"`
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
}

func toUpsertRPCInstanceResource(mesh, _, nodeData string) coremodel.Resource {
	zkInstance := &ZooKeeperInstance{}
	err := json.Unmarshal([]byte(nodeData), zkInstance)
	if err != nil {
		logger.Warnf("parse instance from zookeeper failed, cause: %v, raw content: %s ", err, nodeData)
		return nil
	}
	if strutil.IsBlank(zkInstance.Name) {
		logger.Errorf("invalid content for rpc instance, need app name, raw content: %s", nodeData)
		return nil
	}
	if strutil.IsBlank(zkInstance.Address) {
		logger.Errorf("invalid content for rpc instance, need ip, raw content: %s", nodeData)
		return nil
	}
	if zkInstance.Port <= 0 {
		logger.Errorf("invalid content for rpc instance, need port, raw content: %s", nodeData)
		return nil
	}
	if zkInstance.Payload == nil {
		logger.Errorf("invalid content for rpc instance, need payload, raw content: %s", nodeData)
		return nil
	}
	metadata := zkInstance.Payload.Metadata
	if metadata == nil {
		logger.Errorf("invalid content for rpc instance, need metadata, raw content: %s", nodeData)
		return nil
	}
	return meshresource.ToRPCInstance(mesh, zkInstance.Name, zkInstance.Address, zkInstance.Port, metadata)
}

func toDeleteRPCInstanceResource(mesh, nodePath string) coremodel.Resource {
	paths := strings.Split(nodePath, constants.PathSeparator)
	if len(paths) < 4 {
		return nil
	}
	endpoint := paths[3]
	appName := paths[2]
	splits := strings.Split(endpoint, constants.ColonSeparator)
	ip := splits[0]
	port, err := strconv.ParseInt(splits[1], 10, 64)
	if err != nil {
		logger.Errorf("parse port failed, cause: %v, paths: %s", err, paths)
		return nil
	}
	resName := meshresource.BuildInstanceResName(appName, ip, port)
	return meshresource.NewRPCInstanceResourceWithAttributes(resName, mesh)
}
