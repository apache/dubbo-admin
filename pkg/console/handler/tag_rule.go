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

func TagRuleSearch(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {

		req := model.NewSearchReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		var searchResult *model.SearchPaginationResult
		var err error
		if strutil.IsBlank(req.Keywords) {
			searchResult, err = service.PageListTagRule(ctx, req)
		} else {
			searchResult, err = service.SearchTagRuleByKeywords(ctx, req)
		}
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(searchResult))
	}
}

func GetTagRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		mesh := c.Query("mesh")
		if !strings.HasSuffix(ruleName, constants.TagRuleDotSuffix) {
			err := bizerror.New(bizerror.InvalidArgument, fmt.Sprintf("ruleName must end with %s", constants.TagRuleDotSuffix))
			c.JSON(http.StatusBadRequest, model.NewBizErrorResp(err))
			return
		}
		res, err := service.GetTagRule(ctx, ruleName, mesh)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		if res == nil {
			util.HandleNotFoundError(c, ruleName)
			return
		}
		c.JSON(http.StatusOK, model.GenTagRouteResp(res.Spec))
	}
}

func PutTagRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		mesh := c.Query("mesh")
		if !strings.HasSuffix(ruleName, constants.TagRuleDotSuffix) {
			err := bizerror.New(bizerror.InvalidArgument, fmt.Sprintf("ruleName must end with %s", constants.TagRuleDotSuffix))
			c.JSON(http.StatusBadRequest, model.NewBizErrorResp(err))
			return
		}
		tagRuleRes, err := service.GetTagRule(ctx, ruleName, mesh)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		if tagRuleRes == nil {
			util.HandleNotFoundError(c, ruleName)
			return
		}
		res := meshresource.NewTagRouteResourceWithAttributes(ruleName, mesh)
		err = c.Bind(res.Spec)
		if err != nil {
			c.JSON(http.StatusOK, model.NewErrorResp(err.Error()))
			return
		}
		if err = meshresource.ValidateRule(res); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		opts := mutationOptions(c)
		if err = service.UpdateTagRuleWithOptions(ctx, res, opts); err != nil {
			c.JSON(http.StatusOK, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenTagRouteResp(res.Spec))
		}
	}
}

func PostTagRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		mesh := c.Query("mesh")
		if !strings.HasSuffix(ruleName, constants.TagRuleDotSuffix) {
			err := bizerror.New(bizerror.InvalidArgument, fmt.Sprintf("ruleName must end with %s", constants.TagRuleDotSuffix))
			c.JSON(http.StatusBadRequest, model.NewBizErrorResp(err))
			return
		}
		res := meshresource.NewTagRouteResourceWithAttributes(ruleName, mesh)
		err := c.Bind(res.Spec)
		if err != nil {
			c.JSON(http.StatusOK, model.NewErrorResp(err.Error()))
			return
		}
		if err = meshresource.ValidateRule(res); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		opts := mutationOptions(c)
		if err = service.CreateTagRuleWithOptions(ctx, res, opts); err != nil {
			c.JSON(http.StatusOK, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenTagRouteResp(res.Spec))
		}
	}
}

func DeleteTagRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		mesh := c.Query("mesh")
		if !strings.HasSuffix(ruleName, constants.TagRuleDotSuffix) {
			err := bizerror.New(bizerror.InvalidArgument, fmt.Sprintf("ruleName must end with %s", constants.TagRuleDotSuffix))
			c.JSON(http.StatusBadRequest, model.NewBizErrorResp(err))
			return
		}
		opts := mutationOptions(c)
		if err := service.DeleteTagRuleWithOptions(ctx, ruleName, mesh, opts); err != nil {
			c.JSON(http.StatusOK, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}
