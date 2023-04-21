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

	"github.com/apache/dubbo-admin/pkg/admin/mapper"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/apache/dubbo-admin/pkg/logger"
	"github.com/gin-gonic/gin"
)

var mockRuleService services.MockRuleService = &services.MockRuleServiceImpl{
	MockRuleMapper: &mapper.MockRuleMapperImpl{},
	Logger:         logger.Logger(),
}

func CreateOrUpdateMockRule(c *gin.Context) {
	var mockRule *model.MockRule
	if err := c.ShouldBindJSON(&mockRule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := mockRuleService.CreateOrUpdateMockRule(mockRule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, true)
}

func DeleteMockRuleById(c *gin.Context) {
	// TODO use c.Param("id") instead of http body
	// id, err := strconv.ParseInt(c.Param("id"), 10, 64)

	var mockRule *model.MockRule
	if err := c.ShouldBindJSON(&mockRule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if mockRule.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	if err := mockRuleService.DeleteMockRuleById(int64(mockRule.ID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}

func ListMockRulesByPage(c *gin.Context) {
	filter := c.Query("filter")
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "-1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mockRules, total, err := mockRuleService.ListMockRulesByPage(filter, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// FIXME: the response data is not compatible with the frontend
	c.JSON(http.StatusOK, gin.H{
		"total":   total,
		"content": mockRules,
	})
}
