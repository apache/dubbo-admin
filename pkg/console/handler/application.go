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

	"github.com/gin-gonic/gin"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/core/consts"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
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

		resp, err := service.GetApplicationTabInstanceInfo(ctx, req)
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
		resp, err := service.GetApplicationServiceFormInfo(ctx, req)
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

		resp, err := service.GetApplicationSearchInfo(ctx, req)
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

func generateDefaultConfigurator(name string, scope string, version string, enabled bool) *coremodel.DynamicConfigResource {
	return &coremodel.DynamicConfigResource{
		Meta: nil,
		Spec: &meshproto.DynamicConfig{
			Key:           name,
			Scope:         scope,
			ConfigVersion: version,
			Enabled:       enabled,
			Configs:       make([]*meshproto.OverrideConfig, 0),
		},
	}
}

func ApplicationConfigOperatorLogPut(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			ApplicationName string
			OperatorLog     bool
			isNotExist      = false
		)
		ApplicationName = c.Query("appName")
		OperatorLog, err := strconv.ParseBool(c.Query("operatorLog"))
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		res, err := service.GetConfigurator(ctx, ApplicationName)
		if err != nil {
			if corestore.IsResourceNotFound(err) {
				// for check app exist
				data, err := service.GetApplicationDetail(ctx, &model.ApplicationDetailReq{AppName: ApplicationName})
				if err != nil || data == nil {
					c.JSON(http.StatusNotFound, model.NewErrorResp(err.Error()))
					return
				}
				res = generateDefaultConfigurator(ApplicationName, consts.ScopeApplication, consts.ConfiguratorVersionV3, true)
				isNotExist = true
			} else {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		// append or remove
		if OperatorLog {
			// check is already exist
			alreadyExist := false
			res.Spec.RangeConfig(func(conf *meshproto.OverrideConfig) (isStop bool) {
				alreadyExist = isAppOperatorLogOpened(conf, ApplicationName)
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
				Side:          consts.SideProvider,
				Parameters:    map[string]string{`accesslog`: `true`},
				Enabled:       true,
				Match:         &meshproto.ConditionMatch{Application: &meshproto.ListStringMatch{Oneof: []*meshproto.StringMatch{{Exact: ApplicationName}}}},
				XGenerateByCp: true,
			})
		} else {
			res.Spec.RangeConfigsToRemove(func(conf *meshproto.OverrideConfig) (IsRemove bool) {
				if conf == nil {
					return true
				}
				return isAppOperatorLogOpened(conf, ApplicationName)
			})
		}
		// restore
		if isNotExist {
			err = service.CreateConfigurator(ctx, ApplicationName, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = service.UpdateConfigurator(ctx, ApplicationName, res)
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
		ApplicationName := c.Query("appName")
		if ApplicationName == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("appName is required"))
			return
		}
		res, err := service.GetConfigurator(ctx, ApplicationName)
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
			if isExist = isAppOperatorLogOpened(conf, ApplicationName); isExist {
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
			ApplicationName string
			resp            = struct {
				FlowWeightSets []model.FlowWeightSet `json:"flowWeightSets"`
			}{}
		)
		ApplicationName = c.Query("appName")
		if ApplicationName == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("appName is required"))
			return
		}

		resp.FlowWeightSets = make([]model.FlowWeightSet, 0)

		res, err := service.GetConfigurator(ctx, ApplicationName)
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
			ApplicationName = ""
			body            = struct {
				FlowWeightSets []model.FlowWeightSet `json:"flowWeightSets"`
			}{}
		)
		ApplicationName = c.Query("appName")
		if ApplicationName == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("application name is required"))
		}
		if err := c.Bind(&body); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		// get from store, or generate default resource
		isNotExist := false
		res, err := service.GetConfigurator(ctx, ApplicationName)
		if err != nil {
			if corestore.IsResourceNotFound(err) {
				// for check app exist
				data, err := service.GetApplicationDetail(ctx, &model.ApplicationDetailReq{AppName: ApplicationName})
				if err != nil {
					c.JSON(http.StatusNotFound, model.NewErrorResp(err.Error()))
					return
				} else if data == nil {
					c.JSON(http.StatusNotFound, model.NewErrorResp("application not found"))
					return
				}
				res = generateDefaultConfigurator(ApplicationName, consts.ScopeApplication, consts.ConfiguratorVersionV3, true)
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
			err = service.CreateConfigurator(ctx, ApplicationName, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = service.UpdateConfigurator(ctx, ApplicationName, res)
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
			ApplicationName = ""
			resp            = struct {
				GraySets []model.GraySet `json:"graySets"`
			}{}
		)
		ApplicationName = c.Query("appName")
		if ApplicationName == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("appName is required"))
			return
		}

		res, err := service.GetTagRule(ctx, ApplicationName)
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
			ApplicationName = ""
			body            = struct {
				GraySets []model.GraySet `json:"graySets"`
			}{}
		)
		ApplicationName = c.Query("appName")
		if ApplicationName == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("application name is required"))
			return
		}
		if err := c.Bind(&body); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
		}

		isNotExist := false
		res, err := service.GetTagRule(ctx, ApplicationName)
		if corestore.IsResourceNotFound(err) {
			data, err := service.GetApplicationDetail(ctx, &model.ApplicationDetailReq{AppName: ApplicationName})
			if err != nil {
				c.JSON(http.StatusNotFound, model.NewErrorResp(err.Error()))
				return
			} else if data == nil {
				c.JSON(http.StatusNotFound, model.NewErrorResp("application not found"))
				return
			}
			res = generateDefaultTagRule(ApplicationName, consts.ConfiguratorVersionV3, true, false)
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
			err = service.CreateTagRule(ctx, ApplicationName, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = service.UpdateTagRule(ctx, ApplicationName, res)
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

func generateDefaultTagRule(name string, version string, enable, force bool) *coremodel.TagRouteResource {
	return &coremodel.TagRouteResource{
		Meta: nil,
		Spec: &meshproto.TagRoute{
			Enabled:       enable,
			Key:           name,
			ConfigVersion: version,
			Force:         force,
			Tags:          make([]*meshproto.Tag, 0),
		},
	}
}
