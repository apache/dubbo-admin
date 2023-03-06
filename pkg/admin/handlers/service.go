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

	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/gin-gonic/gin"
)

var providerService services.ProviderService = &services.ProviderServiceImpl{}

func AllServices(c *gin.Context) {
	services := providerService.FindServices()
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": services,
	})
}

func SearchService(c *gin.Context) {
	pattern := c.Query("pattern")
	filter := c.Query("filter")
	providers := providerService.FindService(pattern, filter)
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": providers,
	})
}
