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

package tag_route

import (
	"net/http"
	"strings"

	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/gin-gonic/gin"
)

var tagRouteService services.TagRoutesService = &services.TagRoutesServiceImpl{}

func CreateRule(c *gin.Context) {
	var tagRouteDto model.TagRouteDto
	err := c.BindJSON(&tagRouteDto)
	if err != nil {
		panic(err)
	}

	err = tagRouteService.CreateTagRoute(tagRouteDto)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": "success",
	})
}

func UpdateRule(c *gin.Context) {
	var tagRouteDto model.TagRouteDto
	err := c.BindJSON(&tagRouteDto)
	if err != nil {
		panic(err)
	}
	id := c.Param("id")
	id = strings.ReplaceAll(id, "*", "/")

	_, err = tagRouteService.FindTagRoute(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = tagRouteService.UpdateTagRoute(tagRouteDto)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": "success",
	})
}

func SearchRoutes(c *gin.Context) {
	application := c.Query("application")

	tagRoute, err := tagRouteService.FindTagRoute(application)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": []model.TagRouteDto{tagRoute},
	})
}

func DetailRoute(c *gin.Context) {
	id := c.Param("id")
	id = strings.ReplaceAll(id, "*", "/")

	tagRoute, err := tagRouteService.FindTagRoute(id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": tagRoute,
	})
}

func DeleteRoute(c *gin.Context) {
	id := c.Param("id")
	id = strings.ReplaceAll(id, "*", "/")

	err := tagRouteService.DeleteTagRoute(id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": "success",
	})
}

func EnableRoute(c *gin.Context) {
	id := c.Param("id")
	id = strings.ReplaceAll(id, "*", "/")

	err := tagRouteService.EnableTagRoute(id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": "success",
	})
}

func DisableRoute(c *gin.Context) {
	id := c.Param("id")
	id = strings.ReplaceAll(id, "*", "/")

	err := tagRouteService.DisableTagRoute(id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": "success",
	})
}
