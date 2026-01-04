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

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/console/util"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func ConfiguratorSearch(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewSearchReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		var searchResult *model.SearchPaginationResult
		var err error
		if strutil.IsBlank(req.Keywords) {
			searchResult, err = service.PageListConfiguratorRule(ctx, req)
		} else {
			searchResult, err = service.SearchConfiguratorRuleByKeywords(ctx, req)
		}
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(searchResult))
	}
}

func GetConfiguratorWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		mesh := c.Query("mesh")
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
			util.HandleServiceError(c, err)
			return
		}
		if res == nil {
			c.JSON(http.StatusOK, model.NewBizErrorResp(
				bizerror.New(bizerror.NotFoundError, fmt.Sprintf("%s not found", ruleName))))
		}
		c.JSON(http.StatusOK, model.GenDynamicConfigToResp(res.Spec))
	}
}

func PutConfiguratorWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		mesh := c.Query("mesh")
		if !strings.HasSuffix(ruleName, constants.ConfiguratorRuleDotSuffix) {
			c.JSON(http.StatusOK, model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument,
				fmt.Sprintf("dynamic config name must end with %s", constants.ConfiguratorRuleDotSuffix))))
			return
		}
		res := meshresource.NewDynamicConfigResourceWithAttributes(ruleName, mesh)
		err := c.Bind(res.Spec)
		if err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		configurator, err := service.GetConfigurator(ctx, ruleName, mesh)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		if configurator == nil {
			c.JSON(http.StatusOK, model.NewBizErrorResp(
				bizerror.New(bizerror.NotFoundError, fmt.Sprintf("%s not found", ruleName))))
		}
		if err = service.UpdateConfigurator(ctx, res); err != nil {
			c.JSON(http.StatusOK, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.GenDynamicConfigToResp(res.Spec))
	}
}

func PostConfiguratorWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		mesh := c.Query("mesh")
		if !strings.HasSuffix(ruleName, constants.ConfiguratorRuleDotSuffix) {
			c.JSON(http.StatusBadRequest, model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument,
				fmt.Sprintf("dynamic config name must end with %s", constants.ConfiguratorRuleDotSuffix))))
			return
		}
		res := meshresource.NewDynamicConfigResourceWithAttributes(ruleName, mesh)
		err := c.Bind(res.Spec)
		if err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		if err = service.CreateConfigurator(ctx, res); err != nil {
			c.JSON(http.StatusOK, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.GenDynamicConfigToResp(res.Spec))

	}
}

func DeleteConfiguratorWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		mesh := c.Query("mesh")
		if !strings.HasSuffix(ruleName, constants.ConfiguratorRuleDotSuffix) {
			c.JSON(http.StatusOK, model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument,
				fmt.Sprintf("dynamic config name must end with %s", constants.ConfiguratorRuleDotSuffix))))
			return
		}
		if err := service.DeleteConfigurator(ctx, ruleName, mesh); err != nil {
			c.JSON(http.StatusOK, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}
