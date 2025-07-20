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
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/core/consts"
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

		resp, err := service.GetSearchServices(ctx, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

// service distribution
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

type baseService struct {
	Service string `json:"serviceName"`
	Group   string `json:"group"`
	Version string `json:"version"`
}

func (s *baseService) serviceName() string {
	return s.Service
}

func (s *baseService) query(c *gin.Context) error {
	s.Service = c.Query("serviceName")
	if s.Service == "" {
		return errors.New("service name is empty")
	}
	s.Group = c.Query("group")
	s.Version = c.Query("version")
	return nil
}

func (s *baseService) toInterface() string {
	return s.Service + consts.Colon + s.Group + consts.Colon + s.Version
}

func ServiceConfigTimeoutGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := baseService{}
		resp := struct {
			Timeout int32 `json:"timeout"`
		}{DefaultTimeout}
		if err := param.query(c); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		res, err := service.GetConfigurator(ctx, param.toInterface())
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
			baseService
			Timeout int32 `json:"timeout"`
		}{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		isExist := true
		res, err := service.GetConfigurator(ctx, param.toInterface())
		if err != nil {
			if !corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor) : to check service exist or not
			}
			res = generateDefaultConfigurator(param.serviceName(), consts.ScopeService, consts.ConfiguratorVersionV3, true)
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
			err = service.CreateConfigurator(ctx, param.toInterface(), res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = service.UpdateConfigurator(ctx, param.toInterface(), res)
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
		param := baseService{}
		resp := struct {
			RetryTimes int32 `json:"retryTimes"`
		}{DefaultRetries}
		if err := param.query(c); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		res, err := service.GetConfigurator(ctx, param.toInterface())
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
			baseService
			RetryTimes int32 `json:"retryTimes"`
		}{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		isExist := true
		res, err := service.GetConfigurator(ctx, param.toInterface())
		if err != nil {
			if !corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor) : to check service exist or not
			}
			res = generateDefaultConfigurator(param.serviceName(), consts.ScopeService, consts.ConfiguratorVersionV3, true)
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
			err = service.CreateConfigurator(ctx, param.toInterface(), res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = service.UpdateConfigurator(ctx, param.toInterface(), res)
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
		param := baseService{}
		resp := struct {
			Enabled bool   `json:"enabled"`
			Key     string `json:"key"`
			Ratio   int    `json:"ratio"`
		}{false, "", 0}
		if err := param.query(c); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		res, err := getAffinityRule(ctx, param.toInterface())
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
			baseService
			Enabled bool   `json:"enabled"`
			Key     string `json:"key"`
			Ratio   int    `json:"ratio"`
		}{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		isExist := true
		res, err := getAffinityRule(ctx, param.toInterface())
		if err != nil {
			if !corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor) : to check service exist or not
			} else {
				res = new(mesh.AffinityRouteResource)
				res.Spec = generateDefaultAffinityRule(
					"service",
					param.serviceName(),
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
			err = createAffinityRule(ctx, param.toInterface(), res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = updateAffinityRule(ctx, param.toInterface(), res)
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
			baseService
		}{}
		resp := model.ServiceArgumentRoute{Routes: make([]model.ServiceArgument, 0)}
		if err := param.query(c); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		rawRes, err := service.GetConditionRule(ctx, param.toInterface())
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
			res.RangeConditionsToRemove(func(r *meshproto.ConditionRule) (isRemove bool) { // 去除非方法匹配项
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
			baseService
			model.ServiceArgumentRoute
		}{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		isExist := true
		rawRes, err := service.GetConditionRule(ctx, param.toInterface())
		if err != nil {
			if !corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor) : to check service exist or not
			}
			rawRes = new(mesh.ConditionRouteResource)
			rawRes.Spec = generateDefaultConditionV3x1(true, false, true, param.serviceName(), consts.ScopeService).ToConditionRoute()
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
			err = service.UpdateConditionRule(ctx, param.toInterface(), rawRes)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = service.CreateConditionRule(ctx, param.toInterface(), rawRes)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}
