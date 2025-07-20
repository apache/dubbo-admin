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

package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/core/consts"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func ConfiguratorSearch(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewSearchConfiguratorReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		ruleList := &mesh.DynamicConfigResourceList{}
		var respList []model.ConfiguratorSearchResp
		if req.Keywords == "" {
			if err := ctx.ResourceManager().List(ctx.AppContext(), ruleList, store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			if err := ctx.ResourceManager().List(ctx.AppContext(), ruleList, store.ListByNameContains(req.Keywords), store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			}
		}
		for _, item := range ruleList.Items {
			respList = append(respList, model.ConfiguratorSearchResp{
				RuleName:   item.Meta.GetName(),
				Scope:      item.Spec.GetScope(),
				CreateTime: item.Meta.GetCreationTime().String(),
				Enabled:    item.Spec.GetEnabled(),
			})
		}
		result := model.NewSearchPaginationResult()
		result.List = respList
		result.PageInfo = &ruleList.Pagination
		c.JSON(http.StatusOK, model.NewSuccessResp(result))
	}
}

func GetConfiguratorWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		if strings.HasSuffix(ruleName, consts.ConfiguratorRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.ConfiguratorRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.ConfiguratorRuleSuffix)))
			return
		}
		res, err := service.GetConfigurator(ctx, name)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.GenDynamicConfigToResp(res.Spec))
	}
}

func PutConfiguratorWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		if strings.HasSuffix(ruleName, consts.ConfiguratorRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.ConfiguratorRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.ConfiguratorRuleSuffix)))
			return
		}
		res := &mesh.DynamicConfigResource{
			Meta: nil,
			Spec: &meshproto.DynamicConfig{},
		}
		err := c.Bind(res.Spec)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		if err = service.UpdateConfigurator(ctx, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenDynamicConfigToResp(res.Spec))
		}
	}
}

func PostConfiguratorWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		if strings.HasSuffix(ruleName, consts.ConfiguratorRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.ConfiguratorRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.ConfiguratorRuleSuffix)))
			return
		}
		res := &mesh.DynamicConfigResource{
			Meta: nil,
			Spec: &meshproto.DynamicConfig{},
		}
		err := c.Bind(res.Spec)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		if err = service.CreateConfigurator(ctx, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenDynamicConfigToResp(res.Spec))
		}
	}
}

func DeleteConfiguratorWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		res := &mesh.DynamicConfigResource{Spec: &meshproto.DynamicConfig{}}
		if strings.HasSuffix(ruleName, consts.ConfiguratorRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.ConfiguratorRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.ConfiguratorRuleSuffix)))
			return
		}
		if err := service.DeleteConfigurator(ctx, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}
