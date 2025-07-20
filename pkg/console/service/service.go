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
	"strconv"

	"dubbo.apache.org/dubbo-go/v3/common/constant"

	"github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func GetServiceTabDistribution(ctx consolectx.Context, req *model.ServiceTabDistributionReq) (*model.SearchPaginationResult, error) {
	manager := ctx.ResourceManager()
	mappingList := &mesh.MappingResourceList{}

	serviceName := req.ServiceName

	if err := manager.List(ctx.AppContext(), mappingList); err != nil {
		return nil, err
	}

	res := make([]*model.ServiceTabDistributionResp, 0)

	for _, mapping := range mappingList.Items {
		// 找到对应serviceName的appNames
		if mapping.Spec.InterfaceName == serviceName {
			for _, appName := range mapping.Spec.ApplicationNames {
				dataplaneList := &mesh.DataplaneResourceList{}
				// 每拿到一个appName，都将对应的实例数据填充进dataplaneList, 再通过dataplane拿到这个appName对应的所有实例
				if err := manager.List(ctx.AppContext(), dataplaneList, store.ListByApplication(appName)); err != nil {
					return nil, err
				}

				// 拿到了appName，接下来从dataplane取实例信息
				for _, dataplane := range dataplaneList.Items {
					metadata := &mesh.MetaDataResource{
						Spec: &v1alpha1.MetaData{},
					}
					if err := manager.Get(ctx.AppContext(), metadata, store.GetByRevision(dataplane.Spec.GetExtensions()[v1alpha1.RevisionLabel]), store.GetByType(dataplane.Spec.GetExtensions()["registry-type"])); err != nil {
						return nil, err
					}
					respItem := &model.ServiceTabDistributionResp{}

					serviceInfos := metadata.GetSpec().(*v1alpha1.MetaData).Services
					var sideServiceInfos []*v1alpha1.ServiceInfo
					for _, serviceInfo := range serviceInfos {
						if serviceInfo.GetParams()[constant.SideKey] == req.Side &&
							serviceInfo.Name == req.ServiceName {
							sideServiceInfos = append(sideServiceInfos, serviceInfo)
						}
					}
					if len(sideServiceInfos) > 0 {
						res = append(res, respItem.FromServiceDataplaneResource(dataplane, metadata, appName, req))
					}
				}
			}
		}
	}

	pagedRes := ToSearchPaginationResult(res, model.ByServiceInstanceName(res), req.PageReq)

	return pagedRes, nil
}

func GetSearchServices(ctx consolectx.Context, req *model.ServiceSearchReq) (*model.SearchPaginationResult, error) {
	if req.Keywords != "" {
		return BannerSearchServices(ctx, req)
	}

	res := make([]*model.ServiceSearchResp, 0)
	serviceMap := make(map[string]*model.ServiceSearch)
	manager := ctx.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(ctx.AppContext(), dataplaneList); err != nil {
		return nil, err
	}
	// 通过dataplane extension字段获取所有revision
	revisions := make(map[string]string, 0)
	for _, dataplane := range dataplaneList.Items {
		rev, ok := dataplane.Spec.GetExtensions()[v1alpha1.RevisionLabel]
		if ok {
			revisions[rev] = dataplane.Spec.GetExtensions()["registry-type"]
		}
	}

	// 遍历 revisions
	for rev, t := range revisions {
		metadata := &mesh.MetaDataResource{
			Spec: &v1alpha1.MetaData{},
		}
		err := manager.Get(ctx.AppContext(), metadata, store.GetByRevision(rev), store.GetByType(t))
		if err != nil {
			return nil, err
		}
		for _, serviceInfo := range metadata.Spec.Services {
			if _, ok := serviceMap[serviceInfo.Name]; ok {
				serviceMap[serviceInfo.Name].FromServiceInfo(serviceInfo)
			} else {
				serviceSearch := model.NewServiceSearch(serviceInfo.Name)
				serviceSearch.FromServiceInfo(serviceInfo)
				serviceMap[serviceInfo.Name] = serviceSearch
			}
		}
	}

	for _, serviceSearch := range serviceMap {
		serviceSearchResp := model.NewServiceSearchResp()
		serviceSearchResp.FromServiceSearch(serviceSearch)
		res = append(res, serviceSearchResp)
	}

	pagedRes := ToSearchPaginationResult(res, model.ByServiceName(res), req.PageReq)
	return pagedRes, nil
}

func BannerSearchServices(ctx consolectx.Context, req *model.ServiceSearchReq) (*model.SearchPaginationResult, error) {
	res := make([]*model.ServiceSearchResp, 0)

	manager := ctx.ResourceManager()
	mappingList := &mesh.MappingResourceList{}

	if req.Keywords != "" {
		if err := manager.List(ctx.AppContext(), mappingList, store.ListByNameContains(req.Keywords)); err != nil {
			return nil, err
		}

		for _, mapping := range mappingList.Items {
			serviceSearchResp := model.NewServiceSearchResp()
			serviceSearchResp.ServiceName = mapping.GetMeta().GetName()
			res = append(res, serviceSearchResp)
		}
	}

	pagedRes := ToSearchPaginationResult(res, model.ByServiceName(res), req.PageReq)

	return pagedRes, nil
}

func ToSearchPaginationResult[T any](services []T, data sort.Interface, req model.PageReq) *model.SearchPaginationResult {
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

	nextOffset := ""
	if paginationEnabled {
		if offset+pageSize < lenFilteredItems { // set new offset only if we did not reach the end of the collection
			nextOffset = strconv.Itoa(offset + req.PageSize)
		}
	}

	res.List = list
	res.PageInfo = &coremodel.Pagination{
		Total:      uint32(lenFilteredItems),
		NextOffset: nextOffset,
	}

	return res
}
