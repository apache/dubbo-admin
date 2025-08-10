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
	"strconv"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common/constant"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/console/constants"
	"github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/store"

	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
)

func GetApplicationDetail(ctx context.Context, req *model.ApplicationDetailReq) (*model.ApplicationDetailResp, error) {
	manager := ctx.ResourceManager()
	instanceList := &mesh.DataplaneResourceList{}

	manager.ListPageByKey()
	if err := manager.List(ctx.AppContext(), instanceList, store.ListByApplication(req.AppName)); err != nil {
		return nil, err
	}

	revisions := make(map[string]*mesh.MetaDataResource, 0)
	applicationDetail := model.NewApplicationDetail()
	for _, dataplane := range instanceList.Items {
		if strings.Split(dataplane.GetMeta().GetName(), constant.KeySeparator)[1] == "0" {
			continue
		}
		rev, ok := dataplane.Spec.GetExtensions()[meshproto.RevisionLabel]
		if ok {
			if metadata, cached := revisions[rev]; !cached {
				metadata = &mesh.MetaDataResource{
					Spec: &meshproto.MetaData{},
				}
				if err := manager.Get(ctx.AppContext(), metadata, store.GetByRevision(rev), store.GetByType(dataplane.Spec.GetExtensions()["registry-type"])); err != nil {
					return nil, err
				}
				revisions[rev] = metadata
				applicationDetail.MergeMetaData(metadata)
			}
		}
		applicationDetail.MergeDataplane(dataplane)
		applicationDetail.GetRegistry(ctx)
	}

	respItem := &model.ApplicationDetailResp{
		AppName: req.AppName,
	}
	respItem = respItem.FromApplicationDetail(applicationDetail)

	return respItem, nil
}

func GetApplicationTabInstanceInfo(ctx context.Context, req *model.ApplicationTabInstanceInfoReq) (*model.SearchPaginationResult, error) {
	manager := ctx.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(ctx.AppContext(), dataplaneList, store.ListByApplication(req.AppName), store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
		return nil, err
	}

	res := model.NewSearchPaginationResult()
	list := make([]*model.ApplicationTabInstanceInfoResp, 0, len(dataplaneList.Items))
	for _, dataplane := range dataplaneList.Items {
		if strings.Split(dataplane.Meta.GetName(), constant.KeySeparator)[1] == "0" {
			continue
		}
		resItem := &model.ApplicationTabInstanceInfoResp{}
		resItem.FromDataplaneResource(dataplane)
		resItem.GetRegistry(ctx)
		list = append(list, resItem)
	}

	res.List = list
	res.PageInfo = &dataplaneList.Pagination

	return res, nil
}

func GetApplicationServiceFormInfo(ctx context.Context, req *model.ApplicationServiceFormReq) (*model.SearchPaginationResult, error) {
	manager := ctx.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(ctx.AppContext(), dataplaneList, store.ListByApplication(req.AppName)); err != nil {
		return nil, err
	}

	res := make([]*model.ApplicationServiceFormResp, 0)
	serviceMap := make(map[string]*model.ApplicationServiceForm)
	revisions := make(map[string]*mesh.MetaDataResource, 0)
	for _, dataplane := range dataplaneList.Items {
		rev, ok := dataplane.Spec.GetExtensions()[meshproto.RevisionLabel]
		if !ok {
			continue
		}

		metadata, cached := revisions[rev]
		if !cached {
			metadata = &mesh.MetaDataResource{
				Spec: &meshproto.MetaData{},
			}
			if err := manager.Get(ctx.AppContext(), metadata, store.GetByRevision(rev), store.GetByType(dataplane.Spec.GetExtensions()["registry-type"])); err != nil {
				return nil, err
			}
			revisions[rev] = metadata
		}

		for _, serviceInfo := range metadata.Spec.Services {
			if serviceInfo.Params[constants.ServiceInfoSide] != req.Side {
				continue
			}
			applicationServiceForm := model.NewApplicationServiceForm(serviceInfo.Name)
			if _, ok := serviceMap[serviceInfo.Name]; !ok {
				serviceMap[serviceInfo.Name] = applicationServiceForm
			}

			if err := applicationServiceForm.FromServiceInfo(serviceInfo); err != nil {
				return nil, err
			}
		}
	}

	for _, applicationServiceForm := range serviceMap {
		applicationServiceFormResp := model.NewApplicationServiceFormResp()
		if err := applicationServiceFormResp.FromApplicationServiceForm(applicationServiceForm); err != nil {
			return nil, err
		}
		res = append(res, applicationServiceFormResp)
	}

	pagedRes := ToSearchPaginationResult(res, model.ByAppServiceFormName(res), req.PageReq)

	return pagedRes, nil
}

func GetApplicationSearchInfo(ctx context.Context, req *model.ApplicationSearchReq) (*model.SearchPaginationResult, error) {
	manager := ctx.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if req.Keywords != "" {
		if err := manager.List(ctx.AppContext(), dataplaneList, store.ListByApplicationContains(req.Keywords)); err != nil {
			return nil, err
		}
	} else {
		if err := manager.List(ctx.AppContext(), dataplaneList); err != nil {
			return nil, err
		}
	}

	res := make([]*model.ApplicationSearchResp, 0)
	appMap := make(map[string]*model.ApplicationSearch)
	for _, dataplane := range dataplaneList.Items {
		if strings.Split(dataplane.GetMeta().GetName(), constant.KeySeparator)[1] == "0" {
			continue
		}
		appName := dataplane.Spec.GetExtensions()[meshproto.Application]
		if _, ok := appMap[appName]; !ok {
			appMap[appName] = model.NewApplicationSearch(appName)
		}
		appMap[appName].MergeDataplane(dataplane)
		appMap[appName].GetRegistry(ctx)
	}

	for appName, search := range appMap {
		applicationSearchResp := &model.ApplicationSearchResp{
			AppName: appName,
		}
		res = append(res, applicationSearchResp.FromApplicationSearch(search))
	}

	pagedRes := ToSearchPaginationResult(res, model.ByAppName(res), req.PageReq)
	return pagedRes, nil
}

func BannerSearchApplications(ctx context.Context, req *model.SearchReq) ([]*model.ApplicationSearchResp, error) {
	manager := ctx.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}
	if req.Keywords != "" {
		if err := manager.List(ctx.AppContext(), dataplaneList, store.ListByApplicationContains(req.Keywords)); err != nil {
			return nil, err
		}
	} else {
		if err := manager.List(ctx.AppContext(), dataplaneList); err != nil {
			return nil, err
		}
	}

	res := make([]*model.ApplicationSearchResp, 0)
	appMap := make(map[string]*model.ApplicationSearch)
	for _, dataplane := range dataplaneList.Items {
		appName := dataplane.Spec.GetExtensions()[meshproto.Application]
		if _, ok := appMap[appName]; !ok {
			appMap[appName] = model.NewApplicationSearch(appName)
		}
		appMap[appName].MergeDataplane(dataplane)
		appMap[appName].GetRegistry(ctx)
	}

	for appName, search := range appMap {
		applicationSearchResp := &model.ApplicationSearchResp{
			AppName: appName,
		}
		res = append(res, applicationSearchResp.FromApplicationSearch(search))
	}
	return res, nil
}
