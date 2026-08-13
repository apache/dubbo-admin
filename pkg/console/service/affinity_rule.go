/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0.
 */

package service

import (
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func SearchAffinityRules(ctx consolectx.Context, req *model.SearchConditionRuleReq) (*model.SearchPaginationResult, error) {
	if strutil.IsNotBlank(req.Keywords) {
		res, exists, err := manager.GetByKey[*meshresource.AffinityRouteResource](ctx.ResourceManager(), meshresource.AffinityRouteKind, coremodel.BuildResourceKey(req.Mesh, req.Keywords))
		if err != nil {
			return nil, err
		}
		if !exists {
			return emptyRuleSearch(req.PageReq), nil
		}
		return singleAffinitySearch(res, req.PageReq), nil
	}
	page, err := manager.PageListByIndexes[*meshresource.AffinityRouteResource](ctx.ResourceManager(), meshresource.AffinityRouteKind, []index.IndexCondition{{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals}}, req.PageReq)
	if err != nil {
		return nil, bizerror.New(bizerror.InternalError, "search affinity rules failed")
	}
	list := slice.Map(page.Data, func(_ int, r *meshresource.AffinityRouteResource) *model.RouterRuleSearchResp {
		return affinitySearchItem(r)
	})
	return &model.SearchPaginationResult{List: list, PageInfo: page.Pagination}, nil
}

func affinitySearchItem(r *meshresource.AffinityRouteResource) *model.RouterRuleSearchResp {
	return &model.RouterRuleSearchResp{CreateTime: r.CreationTimestamp.String(), Enabled: r.Spec.Enabled, RuleName: r.Name, Scope: r.Spec.Scope}
}
func singleAffinitySearch(r *meshresource.AffinityRouteResource, page coremodel.PageReq) *model.SearchPaginationResult {
	return &model.SearchPaginationResult{List: []*model.RouterRuleSearchResp{affinitySearchItem(r)}, PageInfo: coremodel.Pagination{Total: 1, PageSize: page.PageSize, PageOffset: page.PageOffset}}
}
func emptyRuleSearch(page coremodel.PageReq) *model.SearchPaginationResult {
	return &model.SearchPaginationResult{List: nil, PageInfo: coremodel.Pagination{PageSize: page.PageSize, PageOffset: page.PageOffset}}
}

func GetAffinityRule(ctx consolectx.Context, name, mesh string) (*meshresource.AffinityRouteResource, error) {
	r, _, err := manager.GetByKey[*meshresource.AffinityRouteResource](ctx.ResourceManager(), meshresource.AffinityRouteKind, coremodel.BuildResourceKey(mesh, name))
	return r, err
}
func CreateAffinityRule(ctx consolectx.Context, r *meshresource.AffinityRouteResource) error {
	return CreateAffinityRuleWithOptions(ctx, r, RuleMutationOptions{})
}
func CreateAffinityRuleWithOptions(ctx consolectx.Context, r *meshresource.AffinityRouteResource, opts RuleMutationOptions) error {
	return createRule(ctx, r, opts)
}
func UpdateAffinityRule(ctx consolectx.Context, r *meshresource.AffinityRouteResource) error {
	return UpdateAffinityRuleWithOptions(ctx, r, RuleMutationOptions{})
}
func UpdateAffinityRuleWithOptions(ctx consolectx.Context, r *meshresource.AffinityRouteResource, opts RuleMutationOptions) error {
	return updateRule(ctx, r, opts)
}
func DeleteAffinityRule(ctx consolectx.Context, name, mesh string) error {
	return DeleteAffinityRuleWithOptions(ctx, name, mesh, RuleMutationOptions{})
}
func DeleteAffinityRuleWithOptions(ctx consolectx.Context, name, mesh string, opts RuleMutationOptions) error {
	return deleteRule(ctx, RuleRef{Kind: meshresource.AffinityRouteKind, Mesh: mesh, Name: name}, opts)
}
