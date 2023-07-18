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

package handlers

import (
	"net/http"
	"strconv"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/apache/dubbo-admin/pkg/admin/mapper"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/gin-gonic/gin"
)

var mockRuleService services.MockRuleService = &services.MockRuleServiceImpl{
	MockRuleMapper: &mapper.MockRuleMapperImpl{},
	Logger:         logger.Logger(),
}

// CreateOrUpdateMockRule godoc
// @Summary      Create or update MockRule
// @Description  Create or update MockRule
// @Tags         MockRules
// @Accept       json
// @Produce      json
// @Param        env       path  string          false  "environment"       default(dev)
// @Param        mockRule  body  model.MockRule  true   "MockRule"
// @Success      201  {boolean} true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/mock/rule [post]
func CreateOrUpdateMockRule(c *gin.Context) {
	var mockRule *model.MockRule
	if err := c.ShouldBindJSON(&mockRule); err != nil {
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}

	if err := mockRuleService.CreateOrUpdateMockRule(mockRule); err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, true)
}

// DeleteMockRuleById godoc
// @Summary      Delete MockRule by id
// @Description  Delete MockRule by id
// @Tags         MockRules
// @Accept       json
// @Produce      json
// @Param        env      path  string          false  "environment"      default(dev)
// @Param        mockRule body  model.MockRule   true   "MockRule"
// @Success      200  {boolean} true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/mock/rule [delete]
func DeleteMockRuleById(c *gin.Context) {
	// TODO use c.Param("id") instead of http body
	// id, err := strconv.ParseInt(c.Param("id"), 10, 64)

	var mockRule *model.MockRule
	if err := c.ShouldBindJSON(&mockRule); err != nil {
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}
	if mockRule.ID == 0 {
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: "id is required"})
		return
	}
	if err := mockRuleService.DeleteMockRuleById(int64(mockRule.ID)); err != nil {
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}

// ListMockRulesByPage godoc
// @Summary      Get MockRules by page
// @Description  Get MockRules by page
// @Tags         MockRules
// @Accept       json
// @Produce      json
// @Param        env       path      string  false  "environment"       default(dev)
// @Param        filter    query     string  false  "filter condition"
// @Param        offset    query     int     false  "page offset"
// @Param        limit     query     int     false  "page limit"
// @Success      200  {object}  model.ListMockRulesByPage
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/mock/rule/list [get]
func ListMockRulesByPage(c *gin.Context) {
	filter := c.Query("filter")
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "-1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}

	mockRules, total, err := mockRuleService.ListMockRulesByPage(filter, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		return
	}

	// FIXME: the response data is not compatible with the frontend
	c.JSON(http.StatusOK, model.ListMockRulesByPage{
		Total:   total,
		Content: mockRules,
	})
}
