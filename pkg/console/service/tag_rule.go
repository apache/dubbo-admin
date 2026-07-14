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

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func PageListTagRule(ctx consolectx.Context, req *model.SearchReq) (*model.SearchPaginationResult, error) {
	pageData, err := manager.PageListByIndexes[*meshresource.TagRouteResource](
		ctx.ResourceManager(),
		meshresource.TagRouteKind,
		[]index.IndexCondition{
			{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals},
		},
		req.PageReq)
	if err != nil {
		logger.Errorf("search tag rule error: %v", err)
		return nil, bizerror.New(bizerror.InternalError, "search tag rule failed, please try again")
	}
	if pageData.Data == nil || len(pageData.Data) == 0 {
		return &model.SearchPaginationResult{
			List: nil,
			PageInfo: coremodel.Pagination{
				Total:      0,
				PageSize:   req.PageReq.PageSize,
				PageOffset: req.PageReq.PageOffset,
			},
		}, nil
	}
	respList := slice.Map(pageData.Data, func(_ int, item *meshresource.TagRouteResource) *model.TagRuleSearchResp {
		return &model.TagRuleSearchResp{
			CreateTime: "",
			Enabled:    item.Spec.Enabled,
			RuleName:   item.Name,
		}
	})
	return &model.SearchPaginationResult{
		List:     respList,
		PageInfo: pageData.Pagination,
	}, nil
}

// SearchTagRuleByKeywords for now, only accurate search is supported
func SearchTagRuleByKeywords(ctx consolectx.Context, req *model.SearchReq) (*model.SearchPaginationResult, error) {
	resKey := coremodel.BuildResourceKey(req.Mesh, req.Keywords)
	tagRuleRes, exists, err := manager.GetByKey[*meshresource.TagRouteResource](ctx.ResourceManager(), meshresource.TagRouteKind, resKey)
	if err != nil {
		logger.Errorf("search tag rule error: %v", err)
		return nil, bizerror.New(bizerror.InternalError, "search tag rule failed, please try again")
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
		List: []*model.TagRuleSearchResp{
			{
				CreateTime: "",
				Enabled:    tagRuleRes.Spec.Enabled,
				RuleName:   tagRuleRes.Name,
			},
		},
		PageInfo: coremodel.Pagination{
			Total:      1,
			PageSize:   req.PageReq.PageSize,
			PageOffset: req.PageReq.PageOffset,
		},
	}, nil
}

func GetTagRule(ctx consolectx.Context, name string, mesh string) (*meshresource.TagRouteResource, error) {
	res, _, err := manager.GetByKey[*meshresource.TagRouteResource](
		ctx.ResourceManager(),
		meshresource.TagRouteKind,
		coremodel.BuildResourceKey(mesh, name),
	)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func UpdateTagRule(ctx consolectx.Context, res *meshresource.TagRouteResource) error {
	return UpdateTagRuleWithOptions(ctx, res, RuleMutationOptions{})
}

func UpdateTagRuleWithOptions(ctx consolectx.Context, res *meshresource.TagRouteResource, opts RuleMutationOptions) error {
	if err := updateRule(ctx, res, opts); err != nil {
		logger.Warnf("update tag rule %s error: %v", res.Name, err)
		return err
	}
	return nil
}

func CreateTagRule(ctx consolectx.Context, res *meshresource.TagRouteResource) error {
	return CreateTagRuleWithOptions(ctx, res, RuleMutationOptions{})
}

func CreateTagRuleWithOptions(ctx consolectx.Context, res *meshresource.TagRouteResource, opts RuleMutationOptions) error {
	if err := createRule(ctx, res, opts); err != nil {
		logger.Warnf("create tag rule %s error: %v", res.Name, err)
		return err
	}
	return nil
}

func DeleteTagRule(ctx consolectx.Context, name string, mesh string) error {
	return DeleteTagRuleWithOptions(ctx, name, mesh, RuleMutationOptions{})
}

func DeleteTagRuleWithOptions(ctx consolectx.Context, name string, mesh string, opts RuleMutationOptions) error {
	ruleRef := RuleRef{Kind: meshresource.TagRouteKind, Mesh: mesh, Name: name}
	if err := deleteRule(ctx, ruleRef, opts); err != nil {
		logger.Warnf("delete tag rule %s error: %v", name, err)
		return err
	}
	return nil
}
