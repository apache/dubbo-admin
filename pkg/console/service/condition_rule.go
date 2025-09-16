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
	"github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

func SearchConditionRules(ctx context.Context, req *model.SearchConditionRuleReq) (*model.SearchPaginationResult, error) {
	pageData, err := manager.PageSearchResourceByConditions[*meshresource.ConditionRouteResource](
		ctx.ResourceManager(),
		meshresource.ConditionRouteKind,
		[]string{"name=" + req.Keywords},
		req.PageRequest())
	if err != nil {
		return nil, err
	}

	var respList []model.ConditionRuleSearchResp
	for _, item := range pageData.Data {
		if v3 := item.Spec.ToConditionRouteV3(); v3 != nil {
			respList = append(respList, model.ConditionRuleSearchResp{
				RuleName:   item.Name,
				Scope:      v3.GetScope(),
				CreateTime: item.CreationTimestamp.String(),
				Enabled:    v3.GetEnabled(),
			})
		} else if v3x1 := item.Spec.ToConditionRouteV3x1(); v3x1 != nil {
			respList = append(respList, model.ConditionRuleSearchResp{
				RuleName:   item.Name,
				Scope:      v3x1.GetScope(),
				CreateTime: item.CreationTimestamp.String(),
				Enabled:    v3x1.GetEnabled(),
			})
		} else {
			logger.Errorf("Invalid condition route %v", item)
		}
	}
	result := model.NewSearchPaginationResult()
	result.List = respList
	result.PageInfo = pageData.Pagination
	return result, nil
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

func UpdateConditionRule(ctx context.Context, name string, res *meshresource.ConditionRouteResource) error {
	if err := ctx.ResourceManager().Update(res); err != nil {
		logger.Warnf("update %s condition failed with error: %s", name, err.Error())
		return err
	}
	return nil
}

func CreateConditionRule(ctx context.Context, name string, res *meshresource.ConditionRouteResource) error {
	if err := ctx.ResourceManager().Add(res); err != nil {
		logger.Warnf("create %s condition failed with error: %s", name, err.Error())
		return err
	}
	return nil
}

func DeleteConditionRule(ctx context.Context, name string, mesh string) error {
	if err := ctx.ResourceManager().DeleteByKey(meshresource.ConditionRouteKind, coremodel.BuildResourceKey(mesh, name)); err != nil {
		return err
	}
	return nil
}
