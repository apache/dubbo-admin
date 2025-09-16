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
	"net/http"
	"strconv"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/gin-gonic/gin"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/core/consts"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	corestore "github.com/apache/dubbo-admin/pkg/core/store"
)

func GetApplicationDetail(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ApplicationDetailReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp, err := service.GetApplicationDetail(ctx, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetApplicationTabInstanceInfo(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewApplicationTabInstanceInfoReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.GetAppInstanceInfo(ctx, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetApplicationServiceForm(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewApplicationServiceFormReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp, err := service.GetAppServiceInfo(ctx, req)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ApplicationSearch(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewApplicationSearchReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.SearchApplications(ctx, req)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func isAppOperatorLogOpened(conf *meshproto.OverrideConfig, appName string) bool {
	if conf.Side != consts.SideProvider ||
		conf.Parameters == nil ||
		conf.Match == nil ||
		conf.Match.Application == nil ||
		conf.Match.Application.Oneof == nil ||
		len(conf.Match.Application.Oneof) != 1 ||
		conf.Match.Application.Oneof[0].Exact != appName {
		return false
	} else if val, ok := conf.Parameters[`accesslog`]; !ok || val != `true` {
		return false
	}
	return true
}

func ApplicationConfigOperatorLogPut(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			appName         string
			operatorLogOpen bool
			isNotExist      = false
			mesh            string
		)
		appName = c.Query("appName")
		mesh = c.Query("mesh")
		operatorLogOpen, err := strconv.ParseBool(c.Query("operatorLog"))
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		appConfiguratorName := appName + consts.ConfiguratorRuleSuffix
		res, err := service.GetConfigurator(ctx, appConfiguratorName, mesh)
		if err != nil {
			if corestore.IsResourceNotFound(err) {
				// check app exists
				data, err := service.GetApplicationDetail(ctx, &model.ApplicationDetailReq{AppName: appName})
				if err != nil || data == nil {
					c.JSON(http.StatusNotFound, model.NewErrorResp(err.Error()))
					return
				}
				res = meshresource.NewDynamicConfigResourceWithAttributes(appConfiguratorName, mesh)
				res.Spec = &meshproto.DynamicConfig{
					Key:           appName,
					Scope:         consts.ScopeApplication,
					ConfigVersion: consts.ConfiguratorVersionV3,
					Enabled:       true,
					Configs:       make([]*meshproto.OverrideConfig, 0),
				}
				isNotExist = true
			} else {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		// append or remove
		if operatorLogOpen {
			// check is already exist
			alreadyExist := false
			res.Spec.RangeConfig(func(conf *meshproto.OverrideConfig) (isStop bool) {
				alreadyExist = isAppOperatorLogOpened(conf, appName)
				return alreadyExist
			})
			if alreadyExist {
				c.JSON(http.StatusOK, model.NewSuccessResp(nil))
				return
			}
			if res.Spec.Configs == nil {
				res.Spec.Configs = make([]*meshproto.OverrideConfig, 0)
			}
			res.Spec.Configs = append(res.Spec.Configs, &meshproto.OverrideConfig{
				Side:       consts.SideProvider,
				Parameters: map[string]string{`accesslog`: `true`},
				Enabled:    true,
				Match: &meshproto.ConditionMatch{
					Application: &meshproto.ListStringMatch{
						Oneof: []*meshproto.StringMatch{
							{
								Exact: appName,
							},
						}}},
				XGenerateByCp: true,
			})
		} else {
			res.Spec.RangeConfigsToRemove(func(conf *meshproto.OverrideConfig) (IsRemove bool) {
				if conf == nil {
					return true
				}
				return isAppOperatorLogOpened(conf, appName)
			})
		}
		// restore
		if isNotExist {
			err = service.CreateConfigurator(ctx, appName, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = service.UpdateConfigurator(ctx, appName, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}

func ApplicationConfigOperatorLogGet(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		appName := c.Query("appName")
		if appName == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("appName is required"))
			return
		}
		mesh := c.Query("mesh")
		if strutil.IsBlank(mesh) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("mesh is required"))
			return
		}
		appConfiguratorName := appName + consts.ConfiguratorRuleSuffix
		res, err := service.GetConfigurator(ctx, appConfiguratorName, mesh)
		if err != nil {
			if corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusOK, model.NewSuccessResp(map[string]interface{}{"operatorLog": false}))
				return
			}
			c.JSON(http.StatusNotFound, model.NewErrorResp(err.Error()))
			return
		}
		isExist := false
		res.Spec.RangeConfig(func(conf *meshproto.OverrideConfig) (isStop bool) {
			if isExist = isAppOperatorLogOpened(conf, appName); isExist {
				return true
			}
			return false
		})
		c.JSON(http.StatusOK, model.NewSuccessResp(map[string]interface{}{"operatorLog": isExist}))
	}
}

func ApplicationConfigFlowWeightGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			appName string
			mesh    string
			resp    = struct {
				FlowWeightSets []model.FlowWeightSet `json:"flowWeightSets"`
			}{}
		)
		appName = c.Query("appName")
		mesh = c.Query("mesh")
		if appName == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("appName is required"))
			return
		}

		resp.FlowWeightSets = make([]model.FlowWeightSet, 0)
		appConfiguratorName := appName + consts.ConfiguratorRuleSuffix
		res, err := service.GetConfigurator(ctx, appConfiguratorName, mesh)
		if err != nil {
			if corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusOK, model.NewSuccessResp(resp))
				return
			}
			c.JSON(http.StatusNotFound, model.NewErrorResp(err.Error()))
			return
		}

		weight := 0
		res.Spec.RangeConfig(func(conf *meshproto.OverrideConfig) (isStop bool) {
			if isFlowWeight(conf) {
				weight, err = strconv.Atoi(conf.Parameters[`weight`])
				if err != nil {
					logger.Error("parse weight failed", err)
					return true
				}
				scope := make([]model.ParamMatch, 0, len(conf.Match.Param))
				for _, param := range conf.Match.Param {
					scope = append(scope, model.ParamMatch{
						Key:   &param.Key,
						Value: model.StringMatchToModelStringMatch(param.Value),
					})
				}

				resp.FlowWeightSets = append(resp.FlowWeightSets, model.FlowWeightSet{
					Weight: int32(weight),
					Scope:  scope,
				})
			}
			return false
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func isFlowWeight(conf *meshproto.OverrideConfig) bool {
	if conf.Side != consts.SideProvider ||
		conf.Parameters == nil ||
		conf.Match == nil ||
		conf.Match.Param == nil {
		return false
	} else if _, ok := conf.Parameters[`weight`]; !ok {
		return false
	}
	return true
}

func ApplicationConfigFlowWeightPUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			appName string
			mesh    string
			body    = struct {
				FlowWeightSets []model.FlowWeightSet `json:"flowWeightSets"`
			}{}
		)
		appName = c.Query("appName")
		mesh = c.Query("mesh")
		if strutil.IsBlank(appName) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("application name is required"))
		}
		if strutil.IsBlank(mesh) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("mesh is required"))
			return
		}
		if err := c.Bind(&body); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		// get from store, or generate default resource
		isNotExist := false
		appConfiguratorName := appName + consts.ConfiguratorRuleSuffix
		res, err := service.GetConfigurator(ctx, appConfiguratorName, mesh)
		if err != nil {
			if corestore.IsResourceNotFound(err) {
				// for check app exist
				data, err := service.GetApplicationDetail(ctx, &model.ApplicationDetailReq{AppName: appName})
				if err != nil {
					c.JSON(http.StatusNotFound, model.NewErrorResp(err.Error()))
					return
				} else if data == nil {
					c.JSON(http.StatusNotFound, model.NewErrorResp("application not found"))
					return
				}
				res = meshresource.NewDynamicConfigResourceWithAttributes(appConfiguratorName, mesh)
				res.Spec = &meshproto.DynamicConfig{
					Key:           appName,
					Scope:         consts.ScopeApplication,
					ConfigVersion: consts.ConfiguratorVersionV3,
					Enabled:       true,
					Configs:       make([]*meshproto.OverrideConfig, 0),
				}
				isNotExist = true
			} else {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}

		// remove old
		res.Spec.RangeConfigsToRemove(func(conf *meshproto.OverrideConfig) (IsRemove bool) {
			return isFlowWeight(conf)
		})
		// append new
		for _, set := range body.FlowWeightSets {
			paramMatch := make([]*meshproto.ParamMatch, 0, len(set.Scope))
			for _, match := range set.Scope {
				paramMatch = append(paramMatch, &meshproto.ParamMatch{
					Key:   *match.Key,
					Value: model.ModelStringMatchToStringMatch(match.Value),
				})
			}
			res.Spec.Configs = append(res.Spec.Configs, &meshproto.OverrideConfig{
				Side:       consts.SideProvider,
				Parameters: map[string]string{`weight`: strconv.Itoa(int(set.Weight))},
				Match: &meshproto.ConditionMatch{
					Param: paramMatch,
				},
				XGenerateByCp: true,
			})
		}
		// restore
		if isNotExist {
			err = service.CreateConfigurator(ctx, appName, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = service.UpdateConfigurator(ctx, appName, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}

func ApplicationConfigGrayGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			appName string
			mesh    string
			resp    = struct {
				GraySets []model.GraySet `json:"graySets"`
			}{}
		)
		appName = c.Query("appName")
		mesh = c.Query("mesh")
		if strutil.IsBlank(appName) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("appName is required"))
			return
		}
		if strutil.IsBlank(mesh) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("mesh is required"))
			return
		}
		serviceTagRuleName := appName + consts.TagRuleSuffix
		res, err := service.GetTagRule(ctx, serviceTagRuleName, mesh)
		if err != nil {
			if corestore.IsResourceNotFound(err) {
				resp.GraySets = make([]model.GraySet, 0)
				c.JSON(http.StatusOK, model.NewSuccessResp(resp))
				return
			}
			c.JSON(http.StatusNotFound, model.NewErrorResp(err.Error()))
			return
		}
		resp.GraySets = make([]model.GraySet, 0, len(res.Spec.Tags))

		res.Spec.RangeTags(func(tag *meshproto.Tag) (isStop bool) {
			if isGrayTag(tag) {
				scope := make([]model.ParamMatch, 0, len(tag.Match))
				for _, paramMatch := range tag.Match {
					scope = append(scope, model.ParamMatch{
						Key:   &paramMatch.Key,
						Value: model.StringMatchToModelStringMatch(paramMatch.Value),
					})
				}
				resp.GraySets = append(resp.GraySets, model.GraySet{
					EnvName: tag.Name,
					Scope:   scope,
				})
			}
			return false
		})

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ApplicationConfigGrayPUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			appName string
			mesh    string
			body    = struct {
				GraySets []model.GraySet `json:"graySets"`
			}{}
		)
		appName = c.Query("appName")
		mesh = c.Query("mesh")
		if strutil.IsBlank(appName) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("application name is required"))
			return
		}
		if strutil.IsBlank(mesh) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("mesh is required"))
			return
		}
		if err := c.Bind(&body); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
		}

		isNotExist := false
		serviceTagRuleName := appName + consts.TagRuleSuffix
		res, err := service.GetTagRule(ctx, serviceTagRuleName, mesh)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
		}
		if res == nil {
			data, err := service.GetApplicationDetail(ctx, &model.ApplicationDetailReq{AppName: appName, Mesh: mesh})
			if err != nil {
				c.JSON(http.StatusNotFound, model.NewErrorResp(err.Error()))
				return
			} else if data == nil {
				c.JSON(http.StatusNotFound, model.NewErrorResp("application not found"))
				return
			}

			res = meshresource.NewTagRouteResourceWithAttributes(serviceTagRuleName, mesh)
			res.Spec = &meshproto.TagRoute{
				Enabled:       true,
				Key:           appName,
				ConfigVersion: consts.ConfiguratorVersionV3,
				Force:         false,
				Tags:          make([]*meshproto.Tag, 0),
			}
			isNotExist = true
		}

		// remove old config, generate config from admin, append
		res.Spec.RangeTagsToRemove(func(tag *meshproto.Tag) (IsRemove bool) {
			return isGrayTag(tag)
		})
		newTags := make([]*meshproto.Tag, 0)
		for _, set := range body.GraySets {
			paramMatches := make([]*meshproto.ParamMatch, 0, len(set.Scope))
			for _, match := range set.Scope {
				paramMatches = append(paramMatches, &meshproto.ParamMatch{
					Key:   *match.Key,
					Value: model.ModelStringMatchToStringMatch(match.Value),
				})
			}
			newTags = append(newTags, &meshproto.Tag{
				Name:          set.EnvName,
				Match:         paramMatches,
				XGenerateByCp: true,
			})
		}
		res.Spec.Tags = append(res.Spec.Tags, newTags...)

		// restore
		if isNotExist {
			err = service.CreateTagRule(ctx, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = service.UpdateTagRule(ctx, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}

func isGrayTag(tag *meshproto.Tag) bool {
	if tag.Name == "" || tag.Addresses != nil || len(tag.Addresses) != 0 {
		return false
	}
	return true
}
