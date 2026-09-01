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
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

// SearchAffinityRules supports exact ruleName lookup and mesh-scoped paging for
// the shared affinity rule list page.
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

// GetAffinityRule loads one affinity rule by the same mesh/name key used by the
// resource store.
func GetAffinityRule(ctx consolectx.Context, name, mesh string) (*meshresource.AffinityRouteResource, error) {
	r, _, err := manager.GetByKey[*meshresource.AffinityRouteResource](ctx.ResourceManager(), meshresource.AffinityRouteKind, coremodel.BuildResourceKey(mesh, name))
	return r, err
}

// CreateAffinityRule preserves the existing no-option service API for callers
// that do not need immediate governor writes.
func CreateAffinityRule(ctx consolectx.Context, r *meshresource.AffinityRouteResource) error {
	return CreateAffinityRuleWithOptions(ctx, r, RuleMutationOptions{})
}

// CreateAffinityRuleWithOptions writes affinity rules through the common rule
// mutation path so ZK/Nacos and local store stay consistent.
func CreateAffinityRuleWithOptions(ctx consolectx.Context, r *meshresource.AffinityRouteResource, opts RuleMutationOptions) error {
	return createRule(ctx, r, opts)
}

// UpdateAffinityRule updates the resource store without extra mutation options.
func UpdateAffinityRule(ctx consolectx.Context, r *meshresource.AffinityRouteResource) error {
	return UpdateAffinityRuleWithOptions(ctx, r, RuleMutationOptions{})
}

// UpdateAffinityRuleWithOptions routes updates through the same governor-aware
// path as condition and tag rules.
func UpdateAffinityRuleWithOptions(ctx consolectx.Context, r *meshresource.AffinityRouteResource, opts RuleMutationOptions) error {
	return updateRule(ctx, r, opts)
}

// DeleteAffinityRule removes one affinity rule without extra mutation options.
func DeleteAffinityRule(ctx consolectx.Context, name, mesh string) error {
	return DeleteAffinityRuleWithOptions(ctx, name, mesh, RuleMutationOptions{})
}

// DeleteAffinityRuleWithOptions deletes the configured rule key from both the
// governor target and local resource state.
func DeleteAffinityRuleWithOptions(ctx consolectx.Context, name, mesh string, opts RuleMutationOptions) error {
	return deleteRule(ctx, RuleRef{Kind: meshresource.AffinityRouteKind, Mesh: mesh, Name: name}, opts)
}
