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

package service

import (
	"sort"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

// GetServiceProviderInstances returns all provider instances for a service so the caller can build a full selector.
func GetServiceProviderInstances(ctx consolectx.Context, req model.BaseServiceReq) ([]*model.SearchInstanceResp, error) {
	metadataList, err := listProviderMeta(ctx, req)
	if err != nil {
		logger.Errorf("list service provider metadata failed, service=%s, mesh=%s, cause: %v", req.ServiceName, req.Mesh, err)
		return nil, err
	}
	if len(metadataList) == 0 {
		return emptyServiceProviderInstancesResult(), nil
	}

	providerAppNames := collectProviderAppNames(metadataList)
	if len(providerAppNames) == 0 {
		return emptyServiceProviderInstancesResult(), nil
	}

	responses := make([]*model.SearchInstanceResp, 0)
	seen := make(map[string]struct{})
	for _, providerAppName := range providerAppNames {
		instanceList, err := manager.ListByIndexes[*meshresource.InstanceResource](
			ctx.ResourceManager(),
			meshresource.InstanceKind,
			map[string]string{
				index.ByMeshIndex:            req.Mesh,
				index.ByInstanceAppNameIndex: providerAppName,
			},
		)
		if err != nil {
			logger.Errorf("list instances failed, service=%s, mesh=%s, providerApp=%s, cause: %v",
				req.ServiceName, req.Mesh, providerAppName, err)
			return nil, err
		}
		for _, instance := range instanceList {
			if instance == nil || instance.Spec == nil {
				continue
			}
			key := instance.ResourceKey()
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			responses = append(responses, model.NewSearchInstanceResp().FromInstanceResource(instance, ctx.Config()))
		}
	}
	if len(responses) == 0 {
		return emptyServiceProviderInstancesResult(), nil
	}

	sort.Slice(responses, func(i, j int) bool {
		if responses[i].AppName != responses[j].AppName {
			return responses[i].AppName < responses[j].AppName
		}
		if responses[i].Name != responses[j].Name {
			return responses[i].Name < responses[j].Name
		}
		return responses[i].Ip < responses[j].Ip
	})

	return responses, nil
}

// collectProviderAppNames extracts unique provider application names from service metadata.
func collectProviderAppNames(metadataList []*meshresource.ServiceProviderMetadataResource) []string {
	providerAppNames := make([]string, 0, len(metadataList))
	seen := make(map[string]struct{}, len(metadataList))
	for _, metadata := range metadataList {
		if metadata == nil || metadata.Spec == nil || metadata.Spec.ProviderAppName == "" {
			continue
		}
		providerAppName := metadata.Spec.ProviderAppName
		if _, exists := seen[providerAppName]; exists {
			continue
		}
		seen[providerAppName] = struct{}{}
		providerAppNames = append(providerAppNames, providerAppName)
	}
	sort.Strings(providerAppNames)
	return providerAppNames
}

func emptyServiceProviderInstancesResult() []*model.SearchInstanceResp {
	return []*model.SearchInstanceResp{}
}
