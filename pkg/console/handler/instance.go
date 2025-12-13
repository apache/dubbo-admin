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
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	corestore "github.com/apache/dubbo-admin/pkg/core/store"
)

func GetInstanceDetail(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.InstanceDetailReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.GetInstanceDetail(ctx, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		if resp == nil {
			c.JSON(http.StatusNotFound, model.NewErrorResp("instance not exist"))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func SearchInstances(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewSearchInstanceReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		instances, err := service.SearchInstances(ctx, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(instances))
	}
}

func InstanceConfigTrafficDisableGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := struct {
			TrafficDisable bool `json:"trafficDisable"`
		}{false}
		applicationName := c.Query("appName")
		mesh := c.Query("mesh")
		if applicationName == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("application name is empty"))
			return
		}
		instanceIP := strings.TrimSpace(c.Query("instanceIP"))
		if instanceIP == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("instanceIP is empty"))
			return
		}

		res, err := service.GetConditionRule(ctx, applicationName, mesh)
		if err != nil {
			if corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusOK, model.NewSuccessResp(resp))
				return
			}
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		if res.Spec.GetVersion() != constants.ConfiguratorVersionV3 {
			c.JSON(http.StatusServiceUnavailable, model.NewErrorResp("this config only serve condition-route.configVersion == v3, got v3.1 config "))
			return
		}

		cr := res.Spec.ToConditionRouteV3()
		cr.RangeConditions(func(condition string) (isStop bool) {
			_, resp.TrafficDisable = isTrafficDisabledV3(condition, instanceIP)
			return resp.TrafficDisable
		})

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func isTrafficDisabledV3X1(r *meshproto.ConditionRule, targetIP string) bool {
	if len(r.To) != 0 {
		return false
	}
	// rule must match `host=x1{,x2,x3}`
	if r.From.Match != "" && !strings.Contains(r.From.Match, "&") && strings.Index(r.From.Match, "!=") == -1 {
		idx := strings.Index(r.From.Match, "=")
		if idx == -1 {
			return false
		}
		then := r.From.Match[idx+1:]
		Ips := strings.Split(then, ",")
		for _, ip := range Ips {
			if strings.TrimSpace(ip) == targetIP {
				return true
			}
		}
	}
	return false
}

/*
*
isTrafficDisabledV3 judge if a condition is disabled or not.
A condition include fromCondition and toCondition which is seperated by `=>`.
The first return parameter `exist` indicates if a condition of specific targetIP exists.
The second return parameter `disabled` indicates if the traffic of targetIP is disabled.
*/
func isTrafficDisabledV3(condition string, targetIP string) (exist bool, disabled bool) {
	if len(condition) == 0 {
		return false, false
	}
	condition = strings.ReplaceAll(condition, " ", "")
	// only accept string start with `=>`
	if !strings.HasPrefix(condition, "=>") {
		return false, false
	}
	toCondition := strings.TrimPrefix(condition, "=>")
	// TODO more specific judge
	if !strings.Contains(toCondition, targetIP) {
		return false, false
	}
	targetExpression := "host!=" + targetIP
	if targetExpression != toCondition {
		return true, false
	}
	return true, true
}

func InstanceConfigTrafficDisablePUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		appName := strings.TrimSpace(c.Query("appName"))
		mesh := c.Query("mesh")
		if appName == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("application name is empty"))
			return
		}
		instanceIP := strings.TrimSpace(c.Query("instanceIP"))
		if instanceIP == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("instanceIP is empty"))
			return
		}
		newDisabled, err := strconv.ParseBool(c.Query(`trafficDisable`))
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(errors.Wrap(err, "parse trafficDisable fail").Error()))
			return
		}

		existRule := true
		rawRes, err := service.GetConditionRule(ctx, appName, mesh)
		var res *meshproto.ConditionRouteV3
		if err != nil {
			if !corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			} else if !newDisabled { // not found && cancel traffic-disable
				c.JSON(http.StatusOK, model.NewSuccessResp(nil))
				return
			}
			existRule = false
			res = generateDefaultConditionV3(true, true, true, appName, constants.ScopeApplication)
			rawRes = meshresource.NewConditionRouteResourceWithAttributes(appName, mesh)
			rawRes.Spec = res.ToConditionRoute()
		} else if res = rawRes.Spec.ToConditionRouteV3(); res == nil {
			c.JSON(http.StatusServiceUnavailable, model.NewErrorResp("this config only serve condition-route.configVersion == v3.1, got v3.0 config "))
			return
		}

		// enable traffic
		if !newDisabled {
			for i, condition := range res.Conditions {
				existCondition, oldDisabled := isTrafficDisabledV3(condition, instanceIP)
				if existCondition {
					if oldDisabled != newDisabled {
						res.Conditions = append(res.Conditions[:i], res.Conditions[i+1:]...)
						rawRes.Spec = res.ToConditionRoute()
						if err = updateORCreateConditionRule(ctx, existRule, appName, rawRes); err != nil {
							c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
						}
						c.JSON(http.StatusOK, model.NewSuccessResp(nil))
						return
					}
				}
			}
		} else { // disable traffic
			// check if condition exists
			for _, condition := range res.Conditions {
				existCondition, oldDisabled := isTrafficDisabledV3(condition, instanceIP)
				if existCondition && oldDisabled {
					c.JSON(http.StatusBadRequest, model.NewErrorResp("The instance has been disabled!"))
					return
				}
			}
			res.Conditions = append(res.Conditions, disableExpression(instanceIP))
			if err = updateORCreateConditionRule(ctx, existRule, appName, rawRes); err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			}
			c.JSON(http.StatusOK, model.NewSuccessResp(nil))
		}
	}
}

func disableExpression(instanceIP string) string {
	return "=>host!=" + instanceIP
}
func updateORCreateConditionRule(ctx consolectx.Context, existRule bool, appName string, rawRes *meshresource.ConditionRouteResource) error {
	if !existRule {
		return service.CreateConditionRule(ctx, appName, rawRes)
	} else {
		return service.UpdateConditionRule(ctx, appName, rawRes)
	}
}

func newDisableConditionV3x1(ip string) *meshproto.ConditionRule {
	return &meshproto.ConditionRule{
		From: &meshproto.ConditionRuleFrom{Match: "host=" + ip},
		To:   nil,
	}
}

func newDisableConditionV3(ip string) string {
	return "=>host!=" + ip
}

func generateDefaultConditionV3x1(Enabled, Force, Runtime bool, Key, Scope string) *meshproto.ConditionRouteV3X1 {
	return &meshproto.ConditionRouteV3X1{
		ConfigVersion: constants.ConfiguratorVersionV3x1,
		Enabled:       Enabled,
		Force:         Force,
		Runtime:       Runtime,
		Key:           Key,
		Scope:         Scope,
		Conditions:    make([]*meshproto.ConditionRule, 0),
	}
}

func generateDefaultConditionV3(Enabled, Force, Runtime bool, Key, Scope string) *meshproto.ConditionRouteV3 {
	return &meshproto.ConditionRouteV3{
		ConfigVersion: constants.ConfiguratorVersionV3,
		Priority:      0,
		Enabled:       true,
		Force:         Force,
		Runtime:       Runtime,
		Key:           Key,
		Scope:         Scope,
		Conditions:    make([]string, 0),
	}
}

func InstanceConfigOperatorLogGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := struct {
			OperatorLog bool `json:"operatorLog"`
		}{false}
		applicationName := c.Query(`appName`)
		if strutil.IsBlank(applicationName) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("application name is empty"))
			return
		}
		instanceIP := c.Query(`instanceIP`)
		if strutil.IsBlank(instanceIP) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("instanceIP is empty"))
			return
		}
		mesh := c.Query(`mesh`)
		if strutil.IsBlank(mesh) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("mesh is empty"))
			return
		}
		appConfiguratorName := applicationName + constants.ConfiguratorRuleSuffix
		res, err := service.GetConfigurator(ctx, appConfiguratorName, mesh)
		if err != nil {
			if corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusOK, model.NewSuccessResp(resp))
				return
			}
			c.JSON(http.StatusNotFound, model.NewErrorResp(err.Error()))
			return
		}

		if res.Spec.Enabled {
			res.Spec.RangeConfig(func(conf *meshproto.OverrideConfig) (isStop bool) {
				resp.OperatorLog = isInstanceOperatorLogOpen(conf, instanceIP)
				return resp.OperatorLog
			})
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func isInstanceOperatorLogOpen(conf *meshproto.OverrideConfig, IP string) bool {
	if conf != nil &&
		conf.Match != nil &&
		conf.Match.Address != nil &&
		conf.Match.Address.Wildcard == IP+`:*` &&
		conf.Side == constants.SideProvider &&
		conf.Parameters != nil &&
		conf.Parameters[`accesslog`] == `true` {
		return true
	}
	return false
}

func InstanceConfigOperatorLogPUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		applicationName := c.Query(`appName`)
		if strutil.IsBlank(applicationName) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("application name is empty"))
			return
		}
		instanceIP := c.Query(`instanceIP`)
		if strutil.IsBlank(instanceIP) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("instanceIP is empty"))
			return
		}
		adminOperatorLog, err := strconv.ParseBool(c.Query(`operatorLog`))
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		mesh := c.Query("mesh")
		if strutil.IsBlank(mesh) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("mesh is empty"))
			return
		}
		appConfiguratorName := applicationName + constants.ConfiguratorRuleSuffix
		res, err := service.GetConfigurator(ctx, appConfiguratorName, mesh)
		notExist := false
		if err != nil {
			if !corestore.IsResourceNotFound(err) {
				c.JSON(http.StatusNotFound, model.NewErrorResp(err.Error()))
				return
			}
			res = meshresource.NewDynamicConfigResourceWithAttributes(appConfiguratorName, mesh)
			res.Spec = &meshproto.DynamicConfig{
				Key:           applicationName,
				Scope:         constants.ScopeApplication,
				ConfigVersion: constants.ConfiguratorVersionV3,
				Enabled:       true,
				Configs:       make([]*meshproto.OverrideConfig, 0),
			}
			notExist = true
		}

		if !adminOperatorLog {
			res.Spec.RangeConfigsToRemove(func(conf *meshproto.OverrideConfig) (IsRemove bool) {
				return isInstanceOperatorLogOpen(conf, instanceIP)
			})
		} else {
			var isExist bool
			res.Spec.RangeConfig(func(conf *meshproto.OverrideConfig) (isStop bool) {
				isExist = isInstanceOperatorLogOpen(conf, instanceIP)
				return isExist
			})
			if !isExist {
				res.Spec.Configs = append(res.Spec.Configs, &meshproto.OverrideConfig{
					Side:          constants.SideProvider,
					Match:         &meshproto.ConditionMatch{Address: &meshproto.AddressMatch{Wildcard: instanceIP + `:*`}},
					Parameters:    map[string]string{`accesslog`: `true`},
					XGenerateByCp: true,
				})
			}
		}

		if notExist {
			err = service.CreateConfigurator(ctx, applicationName, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = service.UpdateConfigurator(ctx, applicationName, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}
