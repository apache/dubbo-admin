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

func PageListConfiguratorRule(ctx consolectx.Context, req *model.SearchReq) (*model.SearchPaginationResult, error) {
	pageData, err := manager.PageListByIndexes[*meshresource.DynamicConfigResource](
		ctx.ResourceManager(),
		meshresource.DynamicConfigKind,
		[]index.IndexCondition{
			{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals},
		},
		req.PageReq)
	if err != nil {
		logger.Errorf("search dynamic config rule error: %v", err)
		return nil, bizerror.New(bizerror.InternalError, "search dynamic config rule failed, please try again")
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
	respList := slice.Map(pageData.Data, func(_ int, item *meshresource.DynamicConfigResource) *model.ConfiguratorSearchResp {
		return &model.ConfiguratorSearchResp{
			Scope:      item.Spec.Scope,
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

// SearchConfiguratorRuleByKeywords for now, only accurate search is supported
func SearchConfiguratorRuleByKeywords(ctx consolectx.Context, req *model.SearchReq) (*model.SearchPaginationResult, error) {
	resKey := coremodel.BuildResourceKey(req.Mesh, req.Keywords)
	configuratorRuleRes, exists, err := manager.GetByKey[*meshresource.DynamicConfigResource](
		ctx.ResourceManager(), meshresource.DynamicConfigKind, resKey)
	if err != nil {
		logger.Errorf("search dynamic config rule error: %v", err)
		return nil, bizerror.New(bizerror.InternalError, "search dynamic config rule failed, please try again")
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
	resp := &model.ConfiguratorSearchResp{
		Scope:      configuratorRuleRes.Spec.Scope,
		CreateTime: "",
		Enabled:    configuratorRuleRes.Spec.Enabled,
		RuleName:   configuratorRuleRes.Name,
	}
	return &model.SearchPaginationResult{
		List: []*model.ConfiguratorSearchResp{resp},
		PageInfo: coremodel.Pagination{
			Total:      1,
			PageSize:   req.PageReq.PageSize,
			PageOffset: req.PageReq.PageOffset,
		},
	}, nil
}

func GetConfigurator(ctx consolectx.Context, name string, mesh string) (*meshresource.DynamicConfigResource, error) {
	res, _, err := manager.GetByKey[*meshresource.DynamicConfigResource](
		ctx.ResourceManager(),
		meshresource.DynamicConfigKind,
		coremodel.BuildResourceKey(mesh, name),
	)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func UpdateConfigurator(ctx consolectx.Context, res *meshresource.DynamicConfigResource) error {
	return UpdateConfiguratorWithOptions(ctx, res, RuleMutationOptions{})
}

func UpdateConfiguratorWithOptions(ctx consolectx.Context, res *meshresource.DynamicConfigResource, opts RuleMutationOptions) error {
	if err := updateRule(ctx, res, opts); err != nil {
		logger.Warnf("update %s configurator failed with error: %s", res.Name, err.Error())
		return err
	}
	return nil
}

func CreateConfigurator(ctx consolectx.Context, res *meshresource.DynamicConfigResource) error {
	return CreateConfiguratorWithOptions(ctx, res, RuleMutationOptions{})
}

func CreateConfiguratorWithOptions(ctx consolectx.Context, res *meshresource.DynamicConfigResource, opts RuleMutationOptions) error {
	if err := createRule(ctx, res, opts); err != nil {
		logger.Warnf("create %s configurator failed with error: %s", res.Name, err.Error())
		return err
	}
	return nil
}

func DeleteConfigurator(ctx consolectx.Context, name string, mesh string) error {
	return DeleteConfiguratorWithOptions(ctx, name, mesh, RuleMutationOptions{})
}

func DeleteConfiguratorWithOptions(ctx consolectx.Context, name string, mesh string, opts RuleMutationOptions) error {
	ruleRef := RuleRef{Kind: meshresource.DynamicConfigKind, Mesh: mesh, Name: name}
	if err := deleteRule(ctx, ruleRef, opts); err != nil {
		logger.Warnf("delete %s configurator failed with error: %s", name, err.Error())
		return err
	}
	return nil
}
