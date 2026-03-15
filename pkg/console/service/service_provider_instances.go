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
	"fmt"
	"sort"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

// GetServiceProviderInstances returns paginated provider instance records for the given service.
func GetServiceProviderInstances(ctx consolectx.Context, req *model.ServiceProviderInstancesReq) (*model.SearchPaginationResult, error) {
	metadataList, err := listServiceProviderMetadata(ctx, model.ServiceMethodsReq{
		ServiceName:     req.ServiceName,
		Group:           req.Group,
		Version:         req.Version,
		Mesh:            req.Mesh,
		ProviderAppName: req.ProviderAppName,
	})
	if err != nil {
		logger.Errorf("list service provider metadata failed, service=%s, mesh=%s, cause: %v", req.ServiceName, req.Mesh, err)
		return nil, err
	}
	if len(metadataList) == 0 {
		return emptyServiceProviderInstancesResult(req), nil
	}

	providerAppNames := collectProviderAppNames(metadataList)
	if len(providerAppNames) == 0 {
		return emptyServiceProviderInstancesResult(req), nil
	}

	responses := make([]*model.ServiceProviderInstanceResp, 0)
	seen := make(map[string]struct{})
	for _, providerAppName := range providerAppNames {
		instanceList, err := manager.ListByIndexes[*meshresource.RPCInstanceResource](
			ctx.ResourceManager(),
			meshresource.RPCInstanceKind,
			map[string]string{
				index.ByMeshIndex:          req.Mesh,
				index.ByRPCInstanceAppName: providerAppName,
			},
		)
		if err != nil {
			logger.Errorf("list rpc instances failed, service=%s, mesh=%s, providerApp=%s, cause: %v",
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
			responses = append(responses, toServiceProviderInstanceResp(instance))
		}
	}
	if len(responses) == 0 {
		return emptyServiceProviderInstancesResult(req), nil
	}

	sort.Slice(responses, func(i, j int) bool {
		if responses[i].AppName != responses[j].AppName {
			return responses[i].AppName < responses[j].AppName
		}
		if responses[i].InstanceName != responses[j].InstanceName {
			return responses[i].InstanceName < responses[j].InstanceName
		}
		if responses[i].IP != responses[j].IP {
			return responses[i].IP < responses[j].IP
		}
		return responses[i].Port < responses[j].Port
	})

	start := req.PageOffset
	if start > len(responses) {
		start = len(responses)
	}
	end := start + req.PageSize
	if end > len(responses) {
		end = len(responses)
	}

	return &model.SearchPaginationResult{
		List: responses[start:end],
		PageInfo: coremodel.Pagination{
			Total:      len(responses),
			PageSize:   req.PageSize,
			PageOffset: req.PageOffset,
		},
	}, nil
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

// toServiceProviderInstanceResp converts an RPC instance resource into an API response model.
func toServiceProviderInstanceResp(instance *meshresource.RPCInstanceResource) *model.ServiceProviderInstanceResp {
	endpoint := ""
	if instance.Spec.GetIp() != "" && instance.Spec.GetPort() > 0 {
		endpoint = fmt.Sprintf("%s:%d", instance.Spec.GetIp(), instance.Spec.GetPort())
	}

	return &model.ServiceProviderInstanceResp{
		AppName:             instance.Spec.GetAppName(),
		InstanceName:        instance.Spec.GetName(),
		IP:                  instance.Spec.GetIp(),
		Port:                instance.Spec.GetPort(),
		Endpoint:            endpoint,
		Protocol:            instance.Spec.GetProtocol(),
		Serialization:       instance.Spec.GetSerialization(),
		PreferSerialization: instance.Spec.GetPreferSerialization(),
		RegisterTime:        instance.Spec.GetRegisterTime(),
	}
}

// emptyServiceProviderInstancesResult builds an empty paginated result for provider instances.
func emptyServiceProviderInstancesResult(req *model.ServiceProviderInstancesReq) *model.SearchPaginationResult {
	return &model.SearchPaginationResult{
		List: []*model.ServiceProviderInstanceResp{},
		PageInfo: coremodel.Pagination{
			Total:      0,
			PageSize:   req.PageSize,
			PageOffset: req.PageOffset,
		},
	}
}
