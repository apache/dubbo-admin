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
	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/consts"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func GetApplicationDetail(ctx consolectx.Context, req *model.ApplicationDetailReq) (*model.ApplicationDetailResp, error) {
	instanceResources, err := manager.ListByIndexes[*meshresource.InstanceResource](
		ctx.ResourceManager(),
		meshresource.InstanceKind,
		map[string]interface{}{
			index.ByMeshIndex:            req.Mesh,
			index.ByInstanceAppNameIndex: req.AppName,
		},
	)
	if err != nil {
		return nil, err
	}

	applicationDetail := model.NewApplicationDetail()
	for _, instanceRes := range instanceResources {
		applicationDetail.MergeInstance(instanceRes)
	}

	respItem := &model.ApplicationDetailResp{
		AppName: req.AppName,
	}
	respItem = respItem.FromApplicationDetail(applicationDetail)

	return respItem, nil
}

func GetAppInstanceInfo(ctx consolectx.Context, req *model.ApplicationTabInstanceInfoReq) (*model.SearchPaginationResult, error) {
	pageData, err := manager.PageListByIndexes[*meshresource.InstanceResource](
		ctx.ResourceManager(),
		meshresource.InstanceKind,
		map[string]interface{}{
			index.ByMeshIndex:            req.Mesh,
			index.ByInstanceAppNameIndex: req.AppName,
		},
		req.PageReq,
	)
	if err != nil {
		return nil, err
	}

	list := slice.Map[*meshresource.InstanceResource, *model.AppInstanceInfoResp](pageData.Data,
		func(_ int, item *meshresource.InstanceResource) *model.AppInstanceInfoResp {
			return buildAppInstanceInfoResp(item)
		})
	searchResult := &model.SearchPaginationResult{
		List:     list,
		PageInfo: pageData.Pagination,
	}
	return searchResult, nil
}

func buildAppInstanceInfoResp(instanceRes *meshresource.InstanceResource) *model.AppInstanceInfoResp {
	instance := instanceRes.Spec
	resp := &model.AppInstanceInfoResp{}
	resp.Name = instance.Name
	resp.AppName = instance.AppName
	resp.CreateTime = instance.CreateTime
	resp.DeployState = instance.DeployState
	resp.DeployClusters = ""
	resp.IP = instance.Ip
	resp.Labels = instance.Tags
	resp.RegisterCluster = instanceRes.Mesh
	if instance.RegisterTime == "" {
		resp.RegisterState = "Registered"
	} else {
		resp.RegisterState = "UnRegistered"
	}
	resp.RegisterTime = instance.RegisterTime
	resp.WorkloadName = instance.WorkloadName
	return resp
}

func GetAppServiceInfo(ctx consolectx.Context, req *model.ApplicationServiceFormReq) (*model.SearchPaginationResult, error) {
	if req.Side == consts.ConsumerSide {
		return getAppProvideServiceInfo(ctx, req)
	} else {
		return getAppConsumeServiceInfo(ctx, req)
	}

}

func getAppProvideServiceInfo(ctx consolectx.Context, req *model.ApplicationServiceFormReq) (*model.SearchPaginationResult, error) {

	pageData, err := manager.PageListByIndexes[*meshresource.ServiceProviderMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceProviderMetadataKind,
		map[string]interface{}{
			index.ByMeshIndex:              req.Mesh,
			index.ByServiceProviderAppName: req.AppName,
		},
		req.PageReq,
	)
	if err != nil {
		return nil, err
	}

	serviceMap := make(map[string]*model.ApplicationServiceFormResp)
	for _, res := range pageData.Data {
		providerMetaData := res.Spec

		if resp, exists := serviceMap[providerMetaData.ServiceName]; exists {
			resp.VersionGroups = append(resp.VersionGroups, model.VersionGroup{
				Version: providerMetaData.Version,
				Group:   providerMetaData.Group,
			})
		} else {
			serviceMap[providerMetaData.ServiceName] = &model.ApplicationServiceFormResp{
				ServiceName: providerMetaData.ServiceName,
				VersionGroups: []model.VersionGroup{
					{
						Version: providerMetaData.Version,
						Group:   providerMetaData.Group,
					},
				},
			}
		}
	}

	pageResult := &model.SearchPaginationResult{
		List:     maputil.Values(serviceMap),
		PageInfo: pageData.Pagination,
	}
	return pageResult, nil
}

func getAppConsumeServiceInfo(ctx consolectx.Context, req *model.ApplicationServiceFormReq) (*model.SearchPaginationResult, error) {
	pageData, err := manager.PageListByIndexes[*meshresource.ServiceConsumerMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceConsumerMetadataKind,
		map[string]interface{}{
			index.ByMeshIndex:              req.Mesh,
			index.ByServiceConsumerAppName: req.AppName,
		},
		req.PageReq,
	)
	if err != nil {
		return nil, err
	}

	serviceMap := make(map[string]*model.ApplicationServiceFormResp)
	for _, res := range pageData.Data {
		consumerMetadata := res.Spec

		if resp, exists := serviceMap[consumerMetadata.ServiceName]; exists {
			resp.VersionGroups = append(resp.VersionGroups, model.VersionGroup{
				Version: consumerMetadata.Version,
				Group:   consumerMetadata.Group,
			})
		} else {
			serviceMap[consumerMetadata.ServiceName] = &model.ApplicationServiceFormResp{
				ServiceName: consumerMetadata.ServiceName,
				VersionGroups: []model.VersionGroup{
					{
						Version: consumerMetadata.Version,
						Group:   consumerMetadata.Group,
					},
				},
			}
		}
	}

	pageResult := &model.SearchPaginationResult{
		List:     maputil.Values(serviceMap),
		PageInfo: pageData.Pagination,
	}
	return pageResult, nil
}

func SearchApplications(ctx consolectx.Context, req *model.ApplicationSearchReq) (*model.SearchPaginationResult, error) {
	pageData, err := searchApplications(ctx, req.AppName, req.Mesh, req.PageReq)
	if err != nil {
		return nil, err
	}
	respList := slice.Map[*meshresource.ApplicationResource, *model.ApplicationSearchResp](pageData.Data,
		func(_ int, item *meshresource.ApplicationResource) *model.ApplicationSearchResp {
			return buildApplicationSearchResp(item, req.Mesh)
		})
	searchResult := &model.SearchPaginationResult{
		List:     respList,
		PageInfo: pageData.Pagination,
	}
	return searchResult, nil
}

func BannerSearchApplications(ctx consolectx.Context, req *model.SearchReq) ([]*model.ApplicationSearchResp, error) {
	pageData, err := searchApplications(ctx, req.Keywords, req.Mesh, req.PageReq)
	if err != nil {
		return nil, err
	}
	respList := slice.Map[*meshresource.ApplicationResource, *model.ApplicationSearchResp](pageData.Data,
		func(_ int, item *meshresource.ApplicationResource) *model.ApplicationSearchResp {
			return buildApplicationSearchResp(item, req.Mesh)
		})
	return respList, nil
}

func searchApplications(
	ctx consolectx.Context,
	keywords string,
	mesh string,
	pageReq coremodel.PageReq) (*coremodel.PageData[*meshresource.ApplicationResource], error) {

	var pageData *coremodel.PageData[*meshresource.ApplicationResource]
	var err error
	if strutil.IsBlank(keywords) {
		pageData, err = manager.PageListByIndexes[*meshresource.ApplicationResource](
			ctx.ResourceManager(),
			meshresource.ApplicationKind,
			map[string]interface{}{
				index.ByMeshIndex: mesh,
			},
			pageReq,
		)
	} else {
		pageData, err = manager.PageSearchResourceByConditions[*meshresource.ApplicationResource](
			ctx.ResourceManager(),
			meshresource.ApplicationKind,
			[]string{"name=" + keywords},
			pageReq,
		)
	}
	if err != nil {
		return nil, err
	}
	return pageData, nil
}

func buildApplicationSearchResp(appResource *meshresource.ApplicationResource, mesh string) *model.ApplicationSearchResp {
	application := appResource.Spec
	return &model.ApplicationSearchResp{
		AppName:          application.Name,
		DeployClusters:   []string{""},
		InstanceCount:    application.InstanceCount,
		RegistryClusters: []string{mesh},
	}
}
