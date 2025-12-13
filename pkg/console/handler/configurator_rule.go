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
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/constants"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func ConfiguratorSearch(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewSearchConfiguratorReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		var pageData *coremodel.PageData[*meshresource.DynamicConfigResource]
		var err error
		if strutil.IsBlank(req.Keywords) {
			pageData, err = manager.PageListByIndexes[*meshresource.DynamicConfigResource](
				ctx.ResourceManager(),
				meshresource.DynamicConfigKind,
				map[string]string{
					index.ByMeshIndex: req.Mesh,
				},
				req.PageReq,
			)

		} else {
			pageData, err = manager.PageSearchResourceByConditions[*meshresource.DynamicConfigResource](
				ctx.ResourceManager(),
				meshresource.DynamicConfigKind,
				[]string{"name=" + req.Keywords},
				req.PageReq,
			)

		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}
		var respList []model.ConfiguratorSearchResp
		for _, res := range pageData.Data {
			configurator := res.Spec
			respList = append(respList, model.ConfiguratorSearchResp{
				RuleName:   configurator.Key,
				Scope:      configurator.Scope,
				CreateTime: "",
				Enabled:    configurator.Enabled,
			})
		}
		result := model.SearchPaginationResult{
			List:     respList,
			PageInfo: pageData.Pagination,
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(result))
	}
}

func GetConfiguratorWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		mesh := c.Param("mesh")
		if strutil.IsBlank(ruleName) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("ruleName cannot be empty"))
			return
		}
		if strutil.IsBlank(mesh) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("mesh cannot be empty"))
			return
		}
		res, err := service.GetConfigurator(ctx, ruleName, mesh)
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
		mesh := c.Param("mesh")
		if strings.HasSuffix(ruleName, constants.ConfiguratorRuleSuffix) {
			name = ruleName[:len(ruleName)-len(constants.ConfiguratorRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", constants.ConfiguratorRuleSuffix)))
			return
		}
		res := meshresource.NewDynamicConfigResourceWithAttributes(name, mesh)
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
		mesh := c.Param("mesh")
		if strings.HasSuffix(ruleName, constants.ConfiguratorRuleSuffix) {
			name = ruleName[:len(ruleName)-len(constants.ConfiguratorRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", constants.ConfiguratorRuleSuffix)))
			return
		}
		res := meshresource.NewDynamicConfigResourceWithAttributes(name, mesh)
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
		mesh := c.Param("mesh")
		if strings.HasSuffix(ruleName, constants.ConfiguratorRuleSuffix) {
			name = ruleName[:len(ruleName)-len(constants.ConfiguratorRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", constants.ConfiguratorRuleSuffix)))
			return
		}
		if err := service.DeleteConfigurator(ctx, name, mesh); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}
