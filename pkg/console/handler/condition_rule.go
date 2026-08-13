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

func ConditionRuleSearch(cs consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewSearchConditionRuleReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp, err := service.SearchConditionRules(cs, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetConditionRuleWithRuleName(cs consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		mesh := c.Query("mesh")
		if !strings.HasSuffix(ruleName, constants.ConditionRuleDotSuffix) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", constants.ConditionRuleDotSuffix)))
			return
		}
		res, err := service.GetConditionRule(cs, ruleName, mesh)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		if res == nil {
			c.JSON(http.StatusOK, model.NewBizErrorResp(
				bizerror.New(bizerror.NotFoundError, fmt.Sprintf("%s not found", ruleName))))
			return
		}
		c.JSON(http.StatusOK, model.GenConditionRuleToResp(res.Spec))
	}
}

func PutConditionRuleWithRuleName(cs consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		mesh := c.Query("mesh")
		if !strings.HasSuffix(ruleName, constants.ConditionRuleDotSuffix) {
			c.JSON(http.StatusBadRequest, model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument,
				fmt.Sprintf("ruleName must end with %s", constants.ConditionRuleDotSuffix))))
			return
		}
		conditionRuleRes, err := service.GetConditionRule(cs, ruleName, mesh)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		if conditionRuleRes == nil {
			util.HandleNotFoundError(c, ruleName)
			return
		}
		res := meshresource.NewConditionRouteResourceWithAttributes(ruleName, mesh)
		input := &model.ConditionRuleInput{}
		if err := c.ShouldBindJSON(input); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		res.Spec, err = input.ToProto()
		if err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		if err := meshresource.ValidateRule(res); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		opts := mutationOptions(c)
		if err := service.UpdateConditionRuleWithOptions(cs, res, opts); err != nil {
			util.HandleServiceError(c, err)
			return
		} else {
			c.JSON(http.StatusOK, model.GenConditionRuleToResp(res.Spec))
		}
	}
}

func PostConditionRuleWithRuleName(cs consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		mesh := c.Query("mesh")
		if !strings.HasSuffix(ruleName, constants.ConditionRuleDotSuffix) {
			c.JSON(http.StatusBadRequest, model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument,
				fmt.Sprintf("ruleName must end with %s", constants.ConditionRuleDotSuffix))))
			return
		}
		res := meshresource.NewConditionRouteResourceWithAttributes(ruleName, mesh)
		input := &model.ConditionRuleInput{}
		if err := c.ShouldBindJSON(input); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		var err error
		res.Spec, err = input.ToProto()
		if err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		if err := meshresource.ValidateRule(res); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		opts := mutationOptions(c)
		if err := service.CreateConditionRuleWithOptions(cs, res, opts); err != nil {
			c.JSON(http.StatusOK, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenConditionRuleToResp(res.Spec))
		}
	}
}

func DeleteConditionRuleWithRuleName(cs consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		mesh := c.Query("mesh")
		if !strings.HasSuffix(ruleName, constants.ConditionRuleDotSuffix) {
			c.JSON(http.StatusBadRequest, model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument,
				fmt.Sprintf("ruleName must end with %s", constants.ConditionRuleDotSuffix))))
			return
		}
		opts := mutationOptions(c)
		if err := service.DeleteConditionRuleWithOptions(cs, ruleName, mesh, opts); err != nil {
			c.JSON(http.StatusOK, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}
