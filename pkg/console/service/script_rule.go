/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0.
 */

package service

import (
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
)

func SearchScriptRules(ctx consolectx.Context, req *model.SearchConditionRuleReq) (*model.SearchPaginationResult, error) {
	if strutil.IsNotBlank(req.Keywords) {
		r, ok, err := manager.GetByKey[*meshresource.ScriptRouteResource](ctx.ResourceManager(), meshresource.ScriptRouteKind, coremodel.BuildResourceKey(req.Mesh, req.Keywords))
		if err != nil {
			return nil, err
		}
		if !ok {
			return emptyRuleSearch(req.PageReq), nil
		}
		return &model.SearchPaginationResult{List: []*model.RouterRuleSearchResp{scriptSearchItem(r)}, PageInfo: coremodel.Pagination{Total: 1, PageSize: req.PageSize, PageOffset: req.PageOffset}}, nil
	}
	page, err := manager.PageListByIndexes[*meshresource.ScriptRouteResource](ctx.ResourceManager(), meshresource.ScriptRouteKind, []index.IndexCondition{{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals}}, req.PageReq)
	if err != nil {
		return nil, err
	}
	return &model.SearchPaginationResult{List: slice.Map(page.Data, func(_ int, r *meshresource.ScriptRouteResource) *model.RouterRuleSearchResp {
		return scriptSearchItem(r)
	}), PageInfo: page.Pagination}, nil
}
func scriptSearchItem(r *meshresource.ScriptRouteResource) *model.RouterRuleSearchResp {
	return &model.RouterRuleSearchResp{CreateTime: r.CreationTimestamp.String(), Enabled: r.Spec.Enabled, RuleName: r.Name, Scope: r.Spec.Scope}
}
func GetScriptRule(ctx consolectx.Context, name, mesh string) (*meshresource.ScriptRouteResource, error) {
	r, _, err := manager.GetByKey[*meshresource.ScriptRouteResource](ctx.ResourceManager(), meshresource.ScriptRouteKind, coremodel.BuildResourceKey(mesh, name))
	return r, err
}
func CreateScriptRule(ctx consolectx.Context, r *meshresource.ScriptRouteResource) error {
	return CreateScriptRuleWithOptions(ctx, r, RuleMutationOptions{})
}
func CreateScriptRuleWithOptions(ctx consolectx.Context, r *meshresource.ScriptRouteResource, opts RuleMutationOptions) error {
	return createRule(ctx, r, opts)
}
func UpdateScriptRule(ctx consolectx.Context, r *meshresource.ScriptRouteResource) error {
	return UpdateScriptRuleWithOptions(ctx, r, RuleMutationOptions{})
}
func UpdateScriptRuleWithOptions(ctx consolectx.Context, r *meshresource.ScriptRouteResource, opts RuleMutationOptions) error {
	return updateRule(ctx, r, opts)
}
func DeleteScriptRule(ctx consolectx.Context, name, mesh string) error {
	return DeleteScriptRuleWithOptions(ctx, name, mesh, RuleMutationOptions{})
}
func DeleteScriptRuleWithOptions(ctx consolectx.Context, name, mesh string, opts RuleMutationOptions) error {
	return deleteRule(ctx, RuleRef{Kind: meshresource.ScriptRouteKind, Mesh: mesh, Name: name}, opts)
}
