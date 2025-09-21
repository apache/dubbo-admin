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
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	corestore "github.com/apache/dubbo-admin/pkg/core/store"
)

const (
	DefaultTimeout = 1000
	DefaultRetries = 2
)

func SearchServices(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewServiceSearchReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.SearchServices(ctx, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetServiceTabDistribution(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ServiceTabDistributionReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.GetServiceTabDistribution(ctx, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ListServices(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// req := &model.SearchInstanceReq{}

		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

func GetServiceDetail(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// req := &model.SearchInstanceReq{}

		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

func GetServiceInterfaces(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// req := &model.SearchInstanceReq{}

		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

func ServiceConfigTimeoutGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := model.BaseServiceReq{}
		resp := struct {
			Timeout int32 `json:"timeout"`
		}{DefaultTimeout}
		if err := param.Query(c); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		serviceConfiguratorName := param.ServiceKey() + consts.PunctuationPoint + consts.ConfiguratorRuleSuffix
		res, err := service.GetConfigurator(ctx, serviceConfiguratorName, param.Mesh)
		if err != nil {
			if !corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor) : to check service exist or not
			}
			c.JSON(http.StatusOK, model.NewSuccessResp(resp))
			return
		}

		res.Spec.RangeConfig(func(conf *meshproto.OverrideConfig) (isStop bool) {
			resp.Timeout, isStop = getServiceTimeout(conf)
			return isStop
		})

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func getServiceTimeout(conf *meshproto.OverrideConfig) (int32, bool) {
	if conf.Side == consts.SideProvider && conf.Parameters != nil && conf.Parameters[`timeout`] != "" {
		timeout, err := strconv.Atoi(conf.Parameters[`timeout`])
		if err == nil {
			return int32(timeout), true
		}
	}
	return DefaultTimeout, false
}

func ServiceConfigTimeoutPUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := struct {
			model.BaseServiceReq
			Timeout int32 `json:"timeout"`
		}{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		isExist := true
		serviceConfiguratorName := param.ServiceKey() + consts.PunctuationPoint + consts.ConfiguratorRuleSuffix
		res, err := service.GetConfigurator(ctx, serviceConfiguratorName, param.Mesh)
		if err != nil {
			if !corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor) : to check service exist or not
			}
			res = meshresource.NewDynamicConfigResourceWithAttributes(serviceConfiguratorName, param.Mesh)
			res.Spec = &meshproto.DynamicConfig{
				Key:           param.ServiceName,
				Scope:         consts.ScopeService,
				ConfigVersion: consts.ConfiguratorVersionV3,
				Enabled:       true,
				Configs:       make([]*meshproto.OverrideConfig, 0),
			}
			isExist = false
		} else {
			res.Spec.RangeConfig(func(conf *meshproto.OverrideConfig) (isStop bool) {
				_, ok := getServiceTimeout(conf)
				if ok {
					conf.Parameters[`timeout`] = strconv.Itoa(int(param.Timeout))
				}
				return ok
			})
		}

		if !isExist {
			res.Spec.Configs = append(res.Spec.Configs, &meshproto.OverrideConfig{
				Side:          consts.SideProvider,
				Parameters:    map[string]string{`timeout`: strconv.Itoa(int(param.Timeout))},
				XGenerateByCp: true,
			})
			err = service.CreateConfigurator(ctx, param.ServiceKey(), res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = service.UpdateConfigurator(ctx, param.ServiceKey(), res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}

func ServiceConfigRetryGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := model.BaseServiceReq{}
		resp := struct {
			RetryTimes int32 `json:"retryTimes"`
		}{DefaultRetries}
		if err := param.Query(c); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		serviceConfiguratorName := param.ServiceKey() + consts.PunctuationPoint + consts.ConfiguratorRuleSuffix
		res, err := service.GetConfigurator(ctx, serviceConfiguratorName, param.Mesh)
		if err != nil {
			if !corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor) : to check service exist or not
			}
			c.JSON(http.StatusOK, model.NewSuccessResp(resp))
			return
		}

		res.Spec.RangeConfig(func(conf *meshproto.OverrideConfig) (isStop bool) {
			resp.RetryTimes, isStop = getServiceRetryTimes(conf)
			return isStop
		})

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func getServiceRetryTimes(conf *meshproto.OverrideConfig) (int32, bool) {
	if conf.Side == consts.SideConsumer && conf.Parameters != nil && conf.Parameters[`retries`] != "" {
		retries, err := strconv.Atoi(conf.Parameters[`retries`])
		if err == nil {
			return int32(retries), true
		}
	}
	return DefaultRetries, false
}

func ServiceConfigRetryPUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := struct {
			model.BaseServiceReq
			RetryTimes int32 `json:"retryTimes"`
		}{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		isExist := true
		serviceConfiguratorName := param.ServiceKey() + consts.PunctuationPoint + consts.ConfiguratorRuleSuffix
		res, err := service.GetConfigurator(ctx, serviceConfiguratorName, param.Mesh)
		if err != nil {
			if !corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor) : to check service exist or not
			}
			res = meshresource.NewDynamicConfigResourceWithAttributes(serviceConfiguratorName, param.Mesh)
			res.Spec = &meshproto.DynamicConfig{
				Key:           param.ServiceName,
				Scope:         consts.ScopeService,
				ConfigVersion: consts.ConfiguratorVersionV3,
				Enabled:       true,
				Configs:       make([]*meshproto.OverrideConfig, 0),
			}
			isExist = false
		}

		res.Spec.RangeConfigsToRemove(func(conf *meshproto.OverrideConfig) (isRemove bool) {
			_, ok := getServiceRetryTimes(conf)
			return ok
		})

		res.Spec.Configs = append(res.Spec.Configs, &meshproto.OverrideConfig{
			Side:          consts.SideConsumer,
			Parameters:    map[string]string{`retries`: strconv.Itoa(int(param.RetryTimes))},
			XGenerateByCp: true,
		})

		if !isExist {
			err = service.CreateConfigurator(ctx, param.ServiceKey(), res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = service.UpdateConfigurator(ctx, param.ServiceKey(), res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}

func ServiceConfigRegionPriorityGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := model.BaseServiceReq{}
		resp := struct {
			Enabled bool   `json:"enabled"`
			Key     string `json:"key"`
			Ratio   int    `json:"ratio"`
		}{false, "", 0}
		if err := param.Query(c); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		serviceAffinityRouteName := param.ServiceKey() + consts.PunctuationPoint + consts.AffinityRuleSuffix
		res, err := service.GetAffinityRule(ctx, serviceAffinityRouteName, param.Mesh)
		if err != nil {
			if !corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor) : to check service exist or not
			}
			c.JSON(http.StatusOK, model.NewSuccessResp(resp))
			return
		} else {
			resp.Enabled = res.Spec.GetEnabled()
			resp.Key = res.Spec.GetAffinity().GetKey()
			resp.Ratio = int(res.Spec.GetAffinity().GetRatio())
			c.JSON(http.StatusOK, model.NewSuccessResp(resp))
			return
		}
	}
}

func ServiceConfigRegionPriorityPUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := struct {
			model.BaseServiceReq
			Enabled bool   `json:"enabled"`
			Key     string `json:"key"`
			Ratio   int    `json:"ratio"`
		}{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		isExist := true
		serviceAffinityRouteName := param.ServiceKey() + consts.PunctuationPoint + consts.AffinityRuleSuffix
		res, err := service.GetAffinityRule(ctx, serviceAffinityRouteName, param.Mesh)
		if err != nil {
			if !corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor) : to check service exist or not
			} else {
				res = meshresource.NewAffinityRouteResourceWithAttributes(serviceAffinityRouteName, param.Mesh)
				res.Spec = generateDefaultAffinityRule(
					"service",
					param.ServiceName,
					param.Key,
					false,
					true,
					param.Ratio,
				)
				isExist = false
			}
		} else {
			res.Spec.Enabled = param.Enabled
			res.Spec.Affinity.Key = param.Key
			res.Spec.Affinity.Ratio = int32(param.Ratio)
		}

		if !isExist {
			err = service.CreateAffinityRule(ctx, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = service.UpdateAffinityRule(ctx, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
		return
	}
}

func generateDefaultAffinityRule(scope, key, focusKey string, runtime, enabled bool, ratio int) *meshproto.AffinityRoute {
	return &meshproto.AffinityRoute{
		ConfigVersion: "v3.1",
		Scope:         scope,
		Key:           key,
		Runtime:       runtime,
		Enabled:       enabled,
		Affinity: &meshproto.AffinityAware{
			Key:   focusKey,
			Ratio: int32(ratio),
		},
	}
}

func ServiceConfigArgumentRouteGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := struct {
			model.BaseServiceReq
		}{}
		resp := model.ServiceArgumentRoute{Routes: make([]model.ServiceArgument, 0)}
		if err := param.Query(c); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		serviceConditionRuleName := param.ServiceKey() + consts.PunctuationPoint + consts.ConditionRuleSuffix
		rawRes, err := service.GetConditionRule(ctx, serviceConditionRuleName, param.Mesh)
		if err != nil {
			if !corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor) : to check service exist or not
			}
			c.JSON(http.StatusOK, model.NewSuccessResp(resp))
			return

		} else if rawRes.Spec.ToConditionRouteV3() != nil {
			c.JSON(http.StatusServiceUnavailable, model.NewErrorResp("this config only serve condition-route.configVersion == v3.1, got v3.0 config "))
			return

		} else {
			res := rawRes.Spec.ToConditionRouteV3x1()
			res.RangeConditionsToRemove(func(r *meshproto.ConditionRule) (isRemove bool) {
				_, ok := r.IsMatchMethod()
				return !ok
			})
			c.JSON(http.StatusOK, model.NewSuccessResp(model.ConditionV3x1ToServiceArgumentRoute(res.Conditions)))
			return
		}
	}
}

func ServiceConfigArgumentRoutePUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := struct {
			model.BaseServiceReq
			model.ServiceArgumentRoute
		}{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		isExist := true
		serviceConditionRuleName := param.ServiceKey() + consts.PunctuationPoint + consts.ConditionRuleSuffix
		rawRes, err := service.GetConditionRule(ctx, serviceConditionRuleName, param.Mesh)
		if err != nil {
			if !corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor) : to check service exist or not
			}
			rawRes = meshresource.NewConditionRouteResourceWithAttributes(serviceConditionRuleName, param.Mesh)
			rawRes.Spec = generateDefaultConditionV3x1(
				true,
				false,
				true,
				param.ServiceName,
				consts.ScopeService).ToConditionRoute()
			isExist = false
		}

		res := rawRes.Spec.ToConditionRouteV3x1()
		if res == nil {
			c.JSON(http.StatusServiceUnavailable, model.NewErrorResp("this config only serve condition-route.configVersion == v3.1, got v3.0 config "))
			return
		}

		if res.Conditions == nil {
			res.Conditions = make([]*meshproto.ConditionRule, 0)
		}
		res.RangeConditionsToRemove(func(r *meshproto.ConditionRule) (isRemove bool) {
			_, ok := r.IsMatchMethod()
			return ok
		})
		res.Conditions = append(res.Conditions, param.ToConditionV3x1Condition()...)
		rawRes.Spec = res.ToConditionRoute()

		if isExist {
			err = service.UpdateConditionRule(ctx, serviceConditionRuleName, rawRes)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = service.CreateConditionRule(ctx, serviceConditionRuleName, rawRes)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}
