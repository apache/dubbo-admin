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

	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/console/util"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func AffinityRuleSearch(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewSearchConditionRuleReq()
		if err := c.ShouldBindQuery(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		result, err := service.SearchAffinityRules(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(result))
	}
}

func GetAffinityRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validRouterRuleName(c, constants.AffinityRuleDotSuffix) {
			return
		}
		name, mesh := c.Param("ruleName"), c.Query("mesh")
		res, err := service.GetAffinityRule(ctx, name, mesh)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		if res == nil {
			util.HandleNotFoundError(c, name)
			return
		}
		c.JSON(http.StatusOK, model.GenAffinityRuleResp(res.Spec))
	}
}

func PostAffinityRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return mutateAffinityRule(ctx, false)
}

func PutAffinityRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return mutateAffinityRule(ctx, true)
}

func mutateAffinityRule(ctx consolectx.Context, update bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validRouterRuleName(c, constants.AffinityRuleDotSuffix) {
			return
		}
		input := &model.AffinityRuleInput{}
		if err := c.ShouldBindJSON(input); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		name, mesh := c.Param("ruleName"), c.Query("mesh")
		res := meshresource.NewAffinityRouteResourceWithAttributes(name, mesh)
		res.Spec = input.ToProto()
		if err := meshresource.ValidateRule(res); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		if update {
			existing, err := service.GetAffinityRule(ctx, name, mesh)
			if err != nil {
				util.HandleServiceError(c, err)
				return
			}
			if existing == nil {
				util.HandleNotFoundError(c, name)
				return
			}
			if err := service.UpdateAffinityRuleWithOptions(ctx, res, mutationOptions(c)); err != nil {
				util.HandleServiceError(c, err)
				return
			}
		} else if err := service.CreateAffinityRuleWithOptions(ctx, res, mutationOptions(c)); err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.GenAffinityRuleResp(res.Spec))
	}
}

func DeleteAffinityRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validRouterRuleName(c, constants.AffinityRuleDotSuffix) {
			return
		}
		if err := service.DeleteAffinityRuleWithOptions(ctx, c.Param("ruleName"), c.Query("mesh"), mutationOptions(c)); err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

func ScriptRuleSearch(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewSearchConditionRuleReq()
		if err := c.ShouldBindQuery(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		result, err := service.SearchScriptRules(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(result))
	}
}

func GetScriptRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validRouterRuleName(c, constants.ScriptRuleDotSuffix) {
			return
		}
		name, mesh := c.Param("ruleName"), c.Query("mesh")
		res, err := service.GetScriptRule(ctx, name, mesh)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		if res == nil {
			util.HandleNotFoundError(c, name)
			return
		}
		c.JSON(http.StatusOK, model.GenScriptRuleResp(res.Spec))
	}
}

func PostScriptRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return mutateScriptRule(ctx, false)
}

func PutScriptRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return mutateScriptRule(ctx, true)
}

func mutateScriptRule(ctx consolectx.Context, update bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validRouterRuleName(c, constants.ScriptRuleDotSuffix) {
			return
		}
		input := &model.ScriptRuleInput{}
		if err := c.ShouldBindJSON(input); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		name, mesh := c.Param("ruleName"), c.Query("mesh")
		res := meshresource.NewScriptRouteResourceWithAttributes(name, mesh)
		res.Spec = input.ToProto()
		if err := meshresource.ValidateRule(res); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		if update {
			existing, err := service.GetScriptRule(ctx, name, mesh)
			if err != nil {
				util.HandleServiceError(c, err)
				return
			}
			if existing == nil {
				util.HandleNotFoundError(c, name)
				return
			}
			if err := service.UpdateScriptRuleWithOptions(ctx, res, mutationOptions(c)); err != nil {
				util.HandleServiceError(c, err)
				return
			}
		} else if err := service.CreateScriptRuleWithOptions(ctx, res, mutationOptions(c)); err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.GenScriptRuleResp(res.Spec))
	}
}

func DeleteScriptRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validRouterRuleName(c, constants.ScriptRuleDotSuffix) {
			return
		}
		if err := service.DeleteScriptRuleWithOptions(ctx, c.Param("ruleName"), c.Query("mesh"), mutationOptions(c)); err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

func validRouterRuleName(c *gin.Context, suffix string) bool {
	if strings.HasSuffix(c.Param("ruleName"), suffix) {
		return true
	}
	err := bizerror.New(bizerror.InvalidArgument, fmt.Sprintf("ruleName must end with %s", suffix))
	c.JSON(http.StatusBadRequest, model.NewBizErrorResp(err))
	return false
}
