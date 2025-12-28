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

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	discoveryutil "github.com/apache/dubbo-admin/pkg/common/util/discovery"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

// GetServiceTabDistribution get service distribution
func GetServiceTabDistribution(ctx consolectx.Context, req *model.ServiceTabDistributionReq) (*model.SearchPaginationResult, error) {
	pageData, err := manager.PageListByIndexes[*meshresource.ServiceConsumerMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceConsumerMetadataKind,
		map[string]string{
			index.ByServiceConsumerServiceName: req.ServiceName,
		}, req.PageReq)
	if err != nil {
		logger.Errorf("get service consumer %s failed, cause: %v", req.ServiceName, err)
		return nil, bizerror.New(bizerror.InternalError, "get service consumer failed, please try again")
	}
	if pageData.Data == nil || len(pageData.Data) == 0 {
		return &model.SearchPaginationResult{
			List: []*meshresource.ServiceConsumerMetadataResourceList{},
			PageInfo: coremodel.Pagination{
				Total:      0,
				PageSize:   req.PageReq.PageSize,
				PageOffset: req.PageReq.PageOffset,
			},
		}, nil
	}
	appNames := slice.Map(pageData.Data, func(_ int, item *meshresource.ServiceConsumerMetadataResource) string {
		return item.Spec.ConsumerAppName
	})
	appResList, err := manager.GetByKeys[*meshresource.ApplicationResource](
		ctx.ResourceManager(), meshresource.ApplicationKind, appNames)
	if err != nil {
		logger.Errorf("get application list %v failed, cause: %s", appNames, err)
		return nil, err
	}
	searchResps := slice.Map(appResList, func(_ int, item *meshresource.ApplicationResource) model.ApplicationSearchResp {
		return model.ApplicationSearchResp{
			AppName:          item.Spec.Name,
			InstanceCount:    item.Spec.InstanceCount,
			DeployClusters:   []string{ctx.Config().Engine.Name},
			RegistryClusters: []string{discoveryutil.GetOrDefaultRegistryName(ctx.Config(), item.Mesh)},
		}
	})
	return &model.SearchPaginationResult{
		List:     searchResps,
		PageInfo: pageData.Pagination,
	}, nil
}

// SearchServices search services pageably
func SearchServices(ctx consolectx.Context, req *model.ServiceSearchReq) (*model.SearchPaginationResult, error) {
	if strutil.IsNotBlank(req.Keywords) {
		return SearchServicesByKeywords(ctx, req)
	}
	pageData, err := manager.PageListByIndexes[*meshresource.ServiceProviderMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceProviderMetadataKind,
		map[string]string{
			index.ByMeshIndex: req.Mesh,
		},
		req.PageReq,
	)
	if err != nil {
		logger.Errorf("get service provider failed, cause: %v", err)
		return nil, err
	}
	if pageData.Data == nil || len(pageData.Data) == 0 {
		return nil, nil
	}
	serviceSearchResps := slice.Map(pageData.Data,
		func(_ int, item *meshresource.ServiceProviderMetadataResource) *model.ServiceSearchResp {
			return ToServiceSearchRespByProvider(item)
		})
	return &model.SearchPaginationResult{
		List:     serviceSearchResps,
		PageInfo: pageData.Pagination,
	}, nil
}

// SearchServicesByKeywords search services by keywords, for now only support accurate search
func SearchServicesByKeywords(ctx consolectx.Context, req *model.ServiceSearchReq) (*model.SearchPaginationResult, error) {
	pageData, err := manager.PageListByIndexes[*meshresource.ServiceProviderMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceProviderMetadataKind,
		map[string]string{
			index.ByServiceProviderServiceName: req.Keywords,
		},
		req.PageReq,
	)
	if err != nil {
		return nil, err
	}
	searchRespList := slice.Map(pageData.Data,
		func(_ int, item *meshresource.ServiceProviderMetadataResource) *model.ServiceSearchResp {
			return ToServiceSearchRespByProvider(item)
		})
	return &model.SearchPaginationResult{
		List:     searchRespList,
		PageInfo: pageData.Pagination,
	}, nil
}

func ToServiceSearchRespByProvider(res *meshresource.ServiceProviderMetadataResource) *model.ServiceSearchResp {
	return &model.ServiceSearchResp{
		ServiceName:     res.Spec.ServiceName,
		Group:           res.Spec.Group,
		Version:         res.Spec.Version,
		ProviderAppName: res.Spec.ProviderAppName,
	}
}

func ToServiceSearchRespByConsumer(res *meshresource.ServiceConsumerMetadataResource) *model.ServiceSearchResp {
	return &model.ServiceSearchResp{
		ServiceName:     res.Spec.ServiceName,
		Group:           res.Spec.Group,
		Version:         res.Spec.Version,
		ConsumerAppName: res.Spec.ConsumerAppName,
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
