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

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/console/util"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

// AffinityRuleSearch lists affinity router rules from the Admin resource store.
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

// GetAffinityRuleWithRuleName returns one affinity rule after validating the
// public ruleName suffix used by Dubbo dynamic configuration.
func GetAffinityRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validRuleSuffix(c, constants.AffinityRuleDotSuffix) {
			return
		}
		r, err := service.GetAffinityRule(ctx, c.Param("ruleName"), c.Query("mesh"))
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		if r == nil {
			util.HandleNotFoundError(c, c.Param("ruleName"))
			return
		}
		c.JSON(http.StatusOK, model.GenAffinityRuleResp(r.Spec))
	}
}
func PostAffinityRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return mutateAffinityRule(ctx, false)
}
func PutAffinityRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return mutateAffinityRule(ctx, true)
}

// mutateAffinityRule shares create/update binding so both endpoints write the
// same public YAML contract through the service layer.
func mutateAffinityRule(ctx consolectx.Context, update bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validRuleSuffix(c, constants.AffinityRuleDotSuffix) {
			return
		}
		input := &model.AffinityRuleInput{}
		if err := c.ShouldBindJSON(input); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		r := meshresource.NewAffinityRouteResourceWithAttributes(c.Param("ruleName"), c.Query("mesh"))
		r.Spec = input.ToProto()
		if err := meshresource.ValidateRule(r); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		var err error
		if update {
			existing, getErr := service.GetAffinityRule(ctx, r.Name, r.Mesh)
			if getErr != nil {
				util.HandleServiceError(c, getErr)
				return
			}
			if existing == nil {
				util.HandleNotFoundError(c, r.Name)
				return
			}
			err = service.UpdateAffinityRuleWithOptions(ctx, r, mutationOptions(c))
		} else {
			err = service.CreateAffinityRuleWithOptions(ctx, r, mutationOptions(c))
		}
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.GenAffinityRuleResp(r.Spec))
	}
}

// DeleteAffinityRuleWithRuleName removes the exact affinity rule key from the
// configured governor so consumers receive the delete event.
func DeleteAffinityRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validRuleSuffix(c, constants.AffinityRuleDotSuffix) {
			return
		}
		if err := service.DeleteAffinityRuleWithOptions(ctx, c.Param("ruleName"), c.Query("mesh"), mutationOptions(c)); err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

// ScriptRuleSearch lists script router rules without exposing script bodies in
// the table response.
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

// GetScriptRuleWithRuleName returns the full script rule spec for editing.
func GetScriptRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validRuleSuffix(c, constants.ScriptRuleDotSuffix) {
			return
		}
		r, err := service.GetScriptRule(ctx, c.Param("ruleName"), c.Query("mesh"))
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		if r == nil {
			util.HandleNotFoundError(c, c.Param("ruleName"))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(r.Spec))
	}
}
func PostScriptRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return mutateScriptRule(ctx, false)
}
func PutScriptRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return mutateScriptRule(ctx, true)
}

// mutateScriptRule keeps script create/update on the same validation path;
// Admin stores the script but never executes it.
func mutateScriptRule(ctx consolectx.Context, update bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validRuleSuffix(c, constants.ScriptRuleDotSuffix) {
			return
		}
		spec := &meshproto.ScriptRoute{}
		if err := c.ShouldBindJSON(spec); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		r := meshresource.NewScriptRouteResourceWithAttributes(c.Param("ruleName"), c.Query("mesh"))
		r.Spec = spec
		if err := meshresource.ValidateRule(r); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		var err error
		if update {
			existing, getErr := service.GetScriptRule(ctx, r.Name, r.Mesh)
			if getErr != nil {
				util.HandleServiceError(c, getErr)
				return
			}
			if existing == nil {
				util.HandleNotFoundError(c, r.Name)
				return
			}
			err = service.UpdateScriptRuleWithOptions(ctx, r, mutationOptions(c))
		} else {
			err = service.CreateScriptRuleWithOptions(ctx, r, mutationOptions(c))
		}
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(r.Spec))
	}
}

// DeleteScriptRuleWithRuleName removes the application-level script rule.
func DeleteScriptRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validRuleSuffix(c, constants.ScriptRuleDotSuffix) {
			return
		}
		if err := service.DeleteScriptRuleWithOptions(ctx, c.Param("ruleName"), c.Query("mesh"), mutationOptions(c)); err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

// validRuleSuffix prevents Admin from publishing keys that Dubbo consumers will
// not subscribe to.
func validRuleSuffix(c *gin.Context, suffix string) bool {
	if strings.HasSuffix(c.Param("ruleName"), suffix) {
		return true
	}
	c.JSON(http.StatusBadRequest, model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument, fmt.Sprintf("ruleName must end with %s", suffix))))
	return false
}
