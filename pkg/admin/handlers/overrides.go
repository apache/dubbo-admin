// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"net/http"

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/apache/dubbo-admin/pkg/admin/util"
	"github.com/gin-gonic/gin"
)

var overrideServiceImpl services.OverrideService = &services.OverrideServiceImpl{
	GovernanceConfig: &config.GovernanceConfigImpl{},
}

func CreateOverride(c *gin.Context) {
	var dynamicConfig *model.DynamicConfig
	if err := c.ShouldBindJSON(&dynamicConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	serviceName := dynamicConfig.Service
	application := dynamicConfig.Application
	if serviceName == "" && application == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service or application must not be empty"})
		return
	}
	// TODO: providerService.findVersionInApplication(application).equals("2.6")
	// if application != "" && providerService.findVersionInApplication(application).equals("2.6") {
	// 	c.JSON(http.StatusBadRequest, errors.New("dubbo 2.6 does not support application scope dynamic config"))
	// 	return
	// }
	err := overrideServiceImpl.SaveOverride(dynamicConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, true)
}

func UpdateOverride(c *gin.Context) {
	id := c.Param("id")
	// env := c.Param("env")
	var dynamicConfig model.DynamicConfig
	if err := c.ShouldBindJSON(&dynamicConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	old, err := overrideServiceImpl.FindOverride(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if old == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "override not found"})
		return
	}
	overrideServiceImpl.UpdateOverride(&dynamicConfig)
	c.JSON(http.StatusOK, true)
}

func SearchOverride(c *gin.Context) {
	service := c.DefaultQuery("service", "")
	application := c.DefaultQuery("application", "")
	serviceVersion := c.DefaultQuery("serviceVersion", "")
	serviceGroup := c.DefaultQuery("serviceGroup", "")

	var override *model.DynamicConfig
	result := make([]*model.DynamicConfig, 0)
	var err error
	if service != "" {
		id := util.BuildServiceKey(service, serviceGroup, serviceVersion)
		override, err = overrideServiceImpl.FindOverride(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if application != "" {
		override, err = overrideServiceImpl.FindOverride(application)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either Service or application is required."})
		return
	}
	if override != nil {
		result = append(result, override)
	}
	c.JSON(http.StatusOK, result)
}

func DetailOverride(c *gin.Context) {
	id := c.Param("id")
	override, err := overrideServiceImpl.FindOverride(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if override == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown ID!"})
		return
	}
	c.JSON(http.StatusOK, override)
}

func EnableOverride(c *gin.Context) {
	id := c.Param("id")
	err := overrideServiceImpl.EnableOverride(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}

func DeleteOverride(c *gin.Context) {
	id := c.Param("id")
	err := overrideServiceImpl.DeleteOverride(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}

func DisableOverride(c *gin.Context) {
	id := c.Param("id")
	err := overrideServiceImpl.DisableOverride(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}
