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
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func GetAffinityRule(ctx consolectx.Context, name, mesh string) (*meshresource.AffinityRouteResource, error) {
	res, _, err := manager.GetByKey[*meshresource.AffinityRouteResource](
		ctx.ResourceManager(), meshresource.AffinityRouteKind, coremodel.BuildResourceKey(mesh, name))
	if err != nil {
		logger.Warnf("get affinity rule %s error: %v", name, err)
		return nil, err
	}
	return res, nil
}

func SearchAffinityRules(ctx consolectx.Context, req *model.SearchConditionRuleReq) (*model.SearchPaginationResult, error) {
	if strutil.IsNotBlank(req.Keywords) {
		res, exists, err := manager.GetByKey[*meshresource.AffinityRouteResource](
			ctx.ResourceManager(), meshresource.AffinityRouteKind,
			coremodel.BuildResourceKey(req.Mesh, req.Keywords))
		if err != nil {
			return nil, err
		}
		if !exists || res == nil {
			return emptyRouterRuleSearch(req.PageReq), nil
		}
		return &model.SearchPaginationResult{
			List:     []*model.RouterRuleSearchResp{affinitySearchItem(res)},
			PageInfo: coremodel.Pagination{Total: 1, PageSize: req.PageSize, PageOffset: req.PageOffset},
		}, nil
	}
	page, err := manager.PageListByIndexes[*meshresource.AffinityRouteResource](
		ctx.ResourceManager(), meshresource.AffinityRouteKind,
		[]index.IndexCondition{{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals}}, req.PageReq)
	if err != nil {
		logger.Errorf("search affinity route error: %v", err)
		return nil, bizerror.New(bizerror.InternalError, "search affinity route failed, please try again")
	}
	list := slice.FilterMap(page.Data, func(_ int, item *meshresource.AffinityRouteResource) (*model.RouterRuleSearchResp, bool) {
		if item == nil || item.Spec == nil {
			return nil, false
		}
		return affinitySearchItem(item), true
	})
	return &model.SearchPaginationResult{List: list, PageInfo: page.Pagination}, nil
}

func affinitySearchItem(r *meshresource.AffinityRouteResource) *model.RouterRuleSearchResp {
	scope := ""
	enabled := false
	if r.Spec != nil {
		scope, enabled = r.Spec.Scope, r.Spec.Enabled
	}
	return &model.RouterRuleSearchResp{CreateTime: r.CreationTimestamp.String(), Enabled: enabled, RuleName: r.Name, Scope: scope}
}

func emptyRouterRuleSearch(page coremodel.PageReq) *model.SearchPaginationResult {
	return &model.SearchPaginationResult{List: nil, PageInfo: coremodel.Pagination{PageSize: page.PageSize, PageOffset: page.PageOffset}}
}

func CreateAffinityRule(ctx consolectx.Context, res *meshresource.AffinityRouteResource) error {
	return CreateAffinityRuleWithOptions(ctx, res, RuleMutationOptions{})
}

func CreateAffinityRuleWithOptions(ctx consolectx.Context, res *meshresource.AffinityRouteResource, opts RuleMutationOptions) error {
	if err := createRule(ctx, res, opts); err != nil {
		logger.Warnf("create %s affinity rule failed with error: %s", res.Name, err.Error())
		return err
	}
	return nil
}

func UpdateAffinityRule(ctx consolectx.Context, res *meshresource.AffinityRouteResource) error {
	return UpdateAffinityRuleWithOptions(ctx, res, RuleMutationOptions{})
}

func UpdateAffinityRuleWithOptions(ctx consolectx.Context, res *meshresource.AffinityRouteResource, opts RuleMutationOptions) error {
	if err := updateRule(ctx, res, opts); err != nil {
		logger.Warnf("update %s affinity rule failed with error: %s", res.Name, err.Error())
		return err
	}
	return nil
}

func DeleteAffinityRule(ctx consolectx.Context, name, mesh string) error {
	return DeleteAffinityRuleWithOptions(ctx, name, mesh, RuleMutationOptions{})
}

func DeleteAffinityRuleWithOptions(ctx consolectx.Context, name, mesh string, opts RuleMutationOptions) error {
	if err := deleteRule(ctx, RuleRef{Kind: meshresource.AffinityRouteKind, Mesh: mesh, Name: name}, opts); err != nil {
		logger.Warnf("delete %s affinity rule failed with error: %s", name, err.Error())
		return err
	}
	return nil
}
