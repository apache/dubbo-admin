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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
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
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetConditionRuleWithRuleName(cs consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		mesh := c.Param("mesh")
		if strings.HasSuffix(ruleName, constants.ConditionRuleDotSuffix) {
			name = ruleName[:len(ruleName)-len(constants.ConditionRuleDotSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", constants.ConditionRuleDotSuffix)))
			return
		}
		if res, err := service.GetConditionRule(cs, name, mesh); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			if v3x1 := res.Spec.ToConditionRouteV3x1(); v3x1 != nil {
				res.Spec = v3x1.ToConditionRoute()
			}
			c.JSON(http.StatusOK, model.GenConditionRuleToResp(res.Spec))
		}
	}
}

func bodyToMap(reader io.ReadCloser) (map[string]interface{}, error) {
	defer reader.Close()
	res := map[string]interface{}{}
	err := json.NewDecoder(reader).Decode(&res)
	return res, err
}

func mapToStructure(m map[string]interface{}, s interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  s,
		TagName: "json",
	})
	if err != nil {
		return err
	}
	err = decoder.Decode(m)
	return err
}

func PutConditionRuleWithRuleName(cs consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		mesh := c.Param("mesh")
		if strings.HasSuffix(ruleName, constants.ConditionRuleDotSuffix) {
			name = ruleName[:len(ruleName)-len(constants.ConditionRuleDotSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", constants.ConditionRuleDotSuffix)))
			return
		}
		_map, err := bodyToMap(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		res := meshresource.NewConditionRouteResourceWithAttributes(ruleName, mesh)
		if version := _map[constants.ConfigVersionKey]; version == constants.ConfiguratorVersionV3 {
			v3 := new(meshproto.ConditionRouteV3)
			err = mapToStructure(_map, &v3)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
			res.Spec = v3.ToConditionRoute()
		} else if version == constants.ConfiguratorVersionV3x1 {
			v3x1 := new(meshproto.ConditionRouteV3X1)
			err = mapToStructure(_map, &v3x1)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}

			res.Spec = v3x1.ToConditionRoute()
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("invalid request body"))
			return
		}

		if err := service.UpdateConditionRule(cs, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenConditionRuleToResp(res.Spec))
		}
	}
}

func PostConditionRuleWithRuleName(cs consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		mesh := c.Param("mesh")
		if strings.HasSuffix(ruleName, constants.ConditionRuleDotSuffix) {
			name = ruleName[:len(ruleName)-len(constants.ConditionRuleDotSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", constants.ConditionRuleDotSuffix)))
			return
		}
		_map, err := bodyToMap(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		res := &meshresource.ConditionRouteResource{}
		res = meshresource.NewConditionRouteResourceWithAttributes(ruleName, mesh)
		if version := _map[constants.ConfigVersionKey]; version == constants.ConfiguratorVersionV3 {
			v3 := new(meshproto.ConditionRouteV3)
			err = mapToStructure(_map, &v3)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
			res.Spec = v3.ToConditionRoute()
		} else if version == constants.ConfiguratorVersionV3x1 {
			v3x1 := new(meshproto.ConditionRouteV3X1)
			err = mapToStructure(_map, &v3x1)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
			res.Spec = v3x1.ToConditionRoute()
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("invalid request body"))
			return
		}

		if err := service.CreateConditionRule(cs, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenConditionRuleToResp(res.Spec))
		}
	}
}

func DeleteConditionRuleWithRuleName(cs consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		mesh := c.Param("mesh")
		if strings.HasSuffix(ruleName, constants.ConditionRuleDotSuffix) {
			name = ruleName[:len(ruleName)-len(constants.ConditionRuleDotSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", constants.ConditionRuleDotSuffix)))
			return
		}
		if err := service.DeleteConditionRule(cs, name, mesh); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}
