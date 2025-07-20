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

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/consts"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func SearchConditionRules(ctx context.Context, req *model.SearchConditionRuleReq) (*model.SearchPaginationResult, error) {
	ruleList := &mesh.ConditionRouteResourceList{}
	if req.Keywords == "" {
		if err := ctx.ResourceManager().List(ctx.AppContext(), ruleList, store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
			return nil, err
		}
	} else {
		if err := ctx.ResourceManager().List(ctx.AppContext(), ruleList, store.ListByNameContains(req.Keywords), store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
			return nil, err
		}
	}

	var respList []model.ConditionRuleSearchResp
	for _, item := range ruleList.Items {
		if v3 := item.Spec.ToConditionRouteV3(); v3 != nil {
			respList = append(respList, model.ConditionRuleSearchResp{
				RuleName:   item.Meta.GetName(),
				Scope:      v3.GetScope(),
				CreateTime: item.Meta.GetCreationTime().String(),
				Enabled:    v3.GetEnabled(),
			})
		} else if v3x1 := item.Spec.ToConditionRouteV3x1(); v3x1 != nil {
			respList = append(respList, model.ConditionRuleSearchResp{
				RuleName:   item.Meta.GetName(),
				Scope:      v3x1.GetScope(),
				CreateTime: item.Meta.GetCreationTime().String(),
				Enabled:    v3x1.GetEnabled(),
			})
		} else {
			logger.Errorf("Invalid condition route %v", item)
		}
	}
	result := model.NewSearchPaginationResult()
	result.List = respList
	result.PageInfo = &ruleList.Pagination
	return result, nil
}

func GetConditionRule(cs context.Context, name string) (*mesh.ConditionRouteResource, error) {
	res := &mesh.ConditionRouteResource{Spec: &meshproto.ConditionRoute{}}
	if err := cs.ResourceManager().Get(cs.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.GetByApplication(name), store.GetByKey(name+consts.ConditionRuleSuffix, coremodel.DefaultMesh)); err != nil {
		logger.Warnf("get %s condition failed with error: %s", name, err.Error())
		return nil, err
	}
	return res, nil
}

func UpdateConditionRule(cs context.Context, name string, res *mesh.ConditionRouteResource) error {
	if err := cs.ResourceManager().Update(cs.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.UpdateByApplication(name), store.UpdateByKey(name+consts.ConditionRuleSuffix, coremodel.DefaultMesh)); err != nil {
		logger.Warnf("update %s condition failed with error: %s", name, err.Error())
		return err
	}
	return nil
}

func CreateConditionRule(cs context.Context, name string, res *mesh.ConditionRouteResource) error {
	if err := cs.ResourceManager().Create(cs.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.CreateByApplication(name), store.CreateByKey(name+consts.ConditionRuleSuffix, coremodel.DefaultMesh)); err != nil {
		logger.Warnf("create %s condition failed with error: %s", name, err.Error())
		return err
	}
	return nil
}

func DeleteConditionRule(cs context.Context, name string, res *mesh.ConditionRouteResource) error {
	if err := cs.ResourceManager().Delete(cs.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.DeleteByApplication(name), store.DeleteByKey(name+consts.ConditionRuleSuffix, coremodel.DefaultMesh)); err != nil {
		logger.Warnf("delete %s condition failed with error: %s", name, err.Error())
		return err
	}
	return nil
}
