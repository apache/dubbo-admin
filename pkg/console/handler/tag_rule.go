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

func TagRuleSearch(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewSearchReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resList := &mesh.TagRouteResourceList{}
		if req.Keywords == "" {
			if err := ctx.ResourceManager().List(ctx.AppContext(), resList, store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			if err := ctx.ResourceManager().List(ctx.AppContext(), resList, store.ListByNameContains(req.Keywords), store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			}
		}
		var respList []model.TagRuleSearchResp
		for _, item := range resList.Items {
			time := item.Meta.GetCreationTime().String()
			name := item.Meta.GetName()
			respList = append(respList, model.TagRuleSearchResp{
				CreateTime: &time,
				Enabled:    &item.Spec.Enabled,
				RuleName:   &name,
			})
		}
		result := model.NewSearchPaginationResult()
		result.List = respList
		result.PageInfo = &resList.Pagination
		c.JSON(http.StatusOK, model.NewSuccessResp(result))
	}
}

func GetTagRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		if strings.HasSuffix(ruleName, consts.TagRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.TagRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.TagRuleSuffix)))
			return
		}
		if res, err := service.GetTagRule(ctx, name); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenTagRouteResp(res.Spec))
		}
	}
}

func PutTagRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		if strings.HasSuffix(ruleName, consts.TagRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.TagRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.TagRuleSuffix)))
			return
		}
		res := &mesh.TagRouteResource{
			Meta: nil,
			Spec: &meshproto.TagRoute{},
		}
		err := c.Bind(res.Spec)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		if err = service.UpdateTagRule(ctx, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenTagRouteResp(res.Spec))
		}
	}
}

func PostTagRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		if strings.HasSuffix(ruleName, consts.TagRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.TagRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.TagRuleSuffix)))
			return
		}
		res := &mesh.TagRouteResource{
			Meta: nil,
			Spec: &meshproto.TagRoute{},
		}
		err := c.Bind(res.Spec)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		if err = service.CreateTagRule(ctx, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenTagRouteResp(res.Spec))
		}
	}
}

func DeleteTagRuleWithRuleName(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		if strings.HasSuffix(ruleName, consts.TagRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.TagRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.TagRuleSuffix)))
			return
		}
		res := &mesh.TagRouteResource{Spec: &meshproto.TagRoute{}}
		if err := service.DeleteTagRule(ctx, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}
