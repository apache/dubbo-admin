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

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func GetScriptRule(ctx consolectx.Context, name, mesh string) (*meshresource.ScriptRouteResource, error) {
	res, _, err := manager.GetByKey[*meshresource.ScriptRouteResource](
		ctx.ResourceManager(), meshresource.ScriptRouteKind, coremodel.BuildResourceKey(mesh, name))
	return res, err
}

func SearchScriptRules(ctx consolectx.Context, req *model.SearchConditionRuleReq) (*model.SearchPaginationResult, error) {
	if strutil.IsNotBlank(req.Keywords) {
		res, exists, err := manager.GetByKey[*meshresource.ScriptRouteResource](
			ctx.ResourceManager(), meshresource.ScriptRouteKind,
			coremodel.BuildResourceKey(req.Mesh, req.Keywords))
		if err != nil {
			return nil, err
		}
		if !exists || res == nil {
			return emptyRouterRuleSearch(req.PageReq), nil
		}
		return &model.SearchPaginationResult{
			List:     []*model.RouterRuleSearchResp{scriptSearchItem(res)},
			PageInfo: coremodel.Pagination{Total: 1, PageSize: req.PageSize, PageOffset: req.PageOffset},
		}, nil
	}
	page, err := manager.PageListByIndexes[*meshresource.ScriptRouteResource](
		ctx.ResourceManager(), meshresource.ScriptRouteKind,
		[]index.IndexCondition{{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals}}, req.PageReq)
	if err != nil {
		return nil, err
	}
	list := slice.FilterMap(page.Data, func(_ int, item *meshresource.ScriptRouteResource) (*model.RouterRuleSearchResp, bool) {
		if item == nil || item.Spec == nil {
			return nil, false
		}
		return scriptSearchItem(item), true
	})
	return &model.SearchPaginationResult{List: list, PageInfo: page.Pagination}, nil
}

func scriptSearchItem(r *meshresource.ScriptRouteResource) *model.RouterRuleSearchResp {
	scope := ""
	enabled := false
	if r.Spec != nil {
		scope, enabled = r.Spec.Scope, r.Spec.Enabled
	}
	return &model.RouterRuleSearchResp{CreateTime: r.CreationTimestamp.String(), Enabled: enabled, RuleName: r.Name, Scope: scope}
}

func CreateScriptRule(ctx consolectx.Context, res *meshresource.ScriptRouteResource) error {
	return CreateScriptRuleWithOptions(ctx, res, RuleMutationOptions{})
}

func CreateScriptRuleWithOptions(ctx consolectx.Context, res *meshresource.ScriptRouteResource, opts RuleMutationOptions) error {
	return createRule(ctx, res, opts)
}

func UpdateScriptRule(ctx consolectx.Context, res *meshresource.ScriptRouteResource) error {
	return UpdateScriptRuleWithOptions(ctx, res, RuleMutationOptions{})
}

func UpdateScriptRuleWithOptions(ctx consolectx.Context, res *meshresource.ScriptRouteResource, opts RuleMutationOptions) error {
	return updateRule(ctx, res, opts)
}

func DeleteScriptRule(ctx consolectx.Context, name, mesh string) error {
	return DeleteScriptRuleWithOptions(ctx, name, mesh, RuleMutationOptions{})
}

func DeleteScriptRuleWithOptions(ctx consolectx.Context, name, mesh string, opts RuleMutationOptions) error {
	return deleteRule(ctx, RuleRef{Kind: meshresource.ScriptRouteKind, Mesh: mesh, Name: name}, opts)
}
