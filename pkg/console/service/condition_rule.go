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
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func SearchConditionRules(ctx context.Context, req *model.SearchConditionRuleReq) (*model.SearchPaginationResult, error) {
	if strutil.IsNotBlank(req.Keywords) {
		return SearchConditionRuleByKeywords(ctx, req)
	}
	pageData, err := manager.PageListByIndexes[*meshresource.ConditionRouteResource](
		ctx.ResourceManager(),
		meshresource.ConditionRouteKind,
		map[string]string{
			index.ByMeshIndex: req.Mesh,
		},
		req.PageReq)
	if err != nil {
		logger.Errorf("search condition route error: %v", err)
		return nil, bizerror.New(bizerror.InternalError, "search condition route failed, please try again")
	}
	respList := slice.FilterMap(pageData.Data,
		func(index int, item *meshresource.ConditionRouteResource) (*model.ConditionRuleSearchResp, bool) {
			resp := ToSearchConditionRuleResp(item)
			return resp, resp != nil
		})
	return &model.SearchPaginationResult{
		List:     respList,
		PageInfo: pageData.Pagination,
	}, nil
}

// SearchConditionRuleByKeywords for now, only accurate search is supported
func SearchConditionRuleByKeywords(ctx context.Context, req *model.SearchConditionRuleReq) (*model.SearchPaginationResult, error) {
	resKey := coremodel.BuildResourceKey(req.Mesh, req.Keywords)
	conditionRuleRes, exists, err := manager.GetByKey[*meshresource.ConditionRouteResource](
		ctx.ResourceManager(), meshresource.ConditionRouteKind, resKey)
	if err != nil {
		logger.Errorf("search condition rule error: %v", err)
		return nil, bizerror.New(bizerror.InternalError, "search condition rule failed, please try again")
	}
	if !exists {
		return &model.SearchPaginationResult{
			List: nil,
			PageInfo: coremodel.Pagination{
				Total:      0,
				PageSize:   req.PageReq.PageSize,
				PageOffset: req.PageReq.PageOffset,
			},
		}, nil
	}
	return &model.SearchPaginationResult{
		List: []*model.ConditionRuleSearchResp{ToSearchConditionRuleResp(conditionRuleRes)},
		PageInfo: coremodel.Pagination{
			Total:      1,
			PageSize:   req.PageReq.PageSize,
			PageOffset: req.PageReq.PageOffset,
		},
	}, nil
}

func ToSearchConditionRuleResp(res *meshresource.ConditionRouteResource) *model.ConditionRuleSearchResp {
	return &model.ConditionRuleSearchResp{
		RuleName:   res.Name,
		Scope:      res.Spec.Scope,
		CreateTime: res.CreationTimestamp.String(),
		Enabled:    res.Spec.Enabled,
	}
}

func GetConditionRule(ctx context.Context, name string, mesh string) (*meshresource.ConditionRouteResource, error) {
	res, _, err := manager.GetByKey[*meshresource.ConditionRouteResource](
		ctx.ResourceManager(), meshresource.ConditionRouteKind, coremodel.BuildResourceKey(mesh, name))
	if err != nil {
		logger.Warnf("get condition route %s error: %v", name, err)
		return nil, err
	}
	return res, nil
}

func UpdateConditionRule(ctx context.Context, res *meshresource.ConditionRouteResource) error {
	if err := ctx.ResourceManager().Update(res); err != nil {
		logger.Warnf("update %s condition failed with error: %s", res.Name, err.Error())
		return err
	}
	return nil
}

func CreateConditionRule(ctx context.Context, res *meshresource.ConditionRouteResource) error {
	if err := ctx.ResourceManager().Add(res); err != nil {
		logger.Warnf("create %s condition failed with error: %s", res.Name, err.Error())
		return err
	}
	return nil
}

func DeleteConditionRule(ctx context.Context, name string, mesh string) error {
	if err := ctx.ResourceManager().DeleteByKey(meshresource.ConditionRouteKind, mesh, coremodel.BuildResourceKey(mesh, name)); err != nil {
		return err
	}
	return nil
}
