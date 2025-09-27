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

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

// GetServiceTabDistribution TODO implement
func GetServiceTabDistribution(ctx consolectx.Context, req *model.ServiceTabDistributionReq) (*model.SearchPaginationResult, error) {
	return nil, nil
}

func SearchServices(ctx consolectx.Context, req *model.ServiceSearchReq) (*model.SearchPaginationResult, error) {
	if req.Keywords != "" {
		return BannerSearchServices(ctx, req)
	}
	var pageData *coremodel.PageData[*meshresource.ServiceResource]
	var err error
	if strutil.IsBlank(req.Keywords) {
		pageData, err = manager.PageListByIndexes[*meshresource.ServiceResource](
			ctx.ResourceManager(),
			meshresource.ServiceKind,
			map[string]string{
				index.ByMeshIndex: req.Mesh,
			},
			req.PageReq,
		)
		if err != nil {
			return nil, err
		}
	} else {
		pageData, err = manager.PageSearchResourceByConditions[*meshresource.ServiceResource](
			ctx.ResourceManager(),
			meshresource.ServiceKind,
			[]string{"serviceName=" + req.Keywords},
			req.PageReq,
		)
		if err != nil {
			return nil, err
		}
	}

	serviceMap := make(map[string]*model.ServiceSearchResp)
	for _, serviceRes := range pageData.Data {
		service := serviceRes.Spec
		if _, exists := serviceMap[service.Name]; !exists {
			serviceMap[service.Name] = toServiceSearchResp(serviceRes)
		} else {
			vgs := serviceMap[service.Name].VersionGroups
			serviceMap[service.Name].VersionGroups = append(vgs, model.VersionGroup{
				Version: service.Version,
				Group:   service.Group,
			})
		}
	}
	return &model.SearchPaginationResult{
		List:     maputil.Values(serviceMap),
		PageInfo: pageData.Pagination,
	}, nil
}

func BannerSearchServices(ctx consolectx.Context, req *model.ServiceSearchReq) (*model.SearchPaginationResult, error) {
	pageData, err := manager.PageSearchResourceByConditions[*meshresource.ServiceResource](
		ctx.ResourceManager(),
		meshresource.ServiceKind,
		[]string{"serviceName=" + req.Keywords},
		req.PageReq,
	)
	if err != nil {
		return nil, err
	}
	searchRespList := slice.Map[*meshresource.ServiceResource, *model.ServiceSearchResp](pageData.Data,
		func(_ int, item *meshresource.ServiceResource) *model.ServiceSearchResp {
			return toServiceSearchResp(item)
		})
	return &model.SearchPaginationResult{
		List:     searchRespList,
		PageInfo: pageData.Pagination,
	}, nil
}

func toServiceSearchResp(res *meshresource.ServiceResource) *model.ServiceSearchResp {
	service := res.Spec
	return &model.ServiceSearchResp{
		ServiceName: service.Name,
		VersionGroups: []model.VersionGroup{
			{
				Version: service.Version,
				Group:   service.Group,
			},
		},
	}

}
func ToSearchPaginationResult[T any](services []T, data sort.Interface, req coremodel.PageReq) *model.SearchPaginationResult {
	res := model.NewSearchPaginationResult()

	list := make([]T, 0)

	sort.Sort(data)
	lenFilteredItems := len(services)
	pageSize := lenFilteredItems
	offset := 0
	paginationEnabled := req.PageSize != 0
	if paginationEnabled {
		pageSize = req.PageSize
		offset = req.PageOffset
	}

	for i := offset; i < offset+pageSize && i < lenFilteredItems; i++ {
		list = append(list, services[i])
	}

	nextOffset := 0
	if paginationEnabled {
		if offset+pageSize < lenFilteredItems { // set new offset only if we did not reach the end of the collection
			nextOffset = offset + req.PageSize
		}
	}

	res.List = list
	res.PageInfo = coremodel.Pagination{
		Total:      lenFilteredItems,
		PageSize:   req.PageSize,
		PageOffset: nextOffset,
	}

	return res
}
