package handlers

import (
	"errors"
	"github.com/apache/dubbo-admin/pkg/admin/model/dto"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/gin-gonic/gin"
	"net/http"
	"reflect"
	"strings"
)

var providerServiceImpl services.ProviderServiceImpl

var overrideServiceImpl services.OverrideServiceImpl

// //var overrideServieImpl services.OverrideService
func CreateOverride(c *gin.Context) {
	var overrideDTO dto.DynamicConfigDTO
	if err := c.BindJSON(&overrideDTO); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	serviceName := overrideDTO.Service
	application := overrideDTO.Application
	if serviceName == "" && application == "" {
		c.AbortWithError(http.StatusBadRequest, errors.New("serviceName and application are Empty!"))
		return
	}
	//TODO providerService.findVersionInApplication(application).equals("2.6")
	if application != "" {

		c.AbortWithError(http.StatusBadRequest, errors.New("dubbo 2.6 does not support application scope dynamic config"))
		return
	}
	overrideServiceImpl.SaveOverride(&overrideDTO)
	c.Status(http.StatusCreated)
}

func UpdateOverride(c *gin.Context) {
	id := c.Param("id")
	//env := c.Param("env")
	var overrideDTO dto.DynamicConfigDTO
	err := c.ShouldBindJSON(&overrideDTO)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	old, err := overrideServiceImpl.FindOverride(id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if reflect.ValueOf(old).IsNil() {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Unknown ID!"})
		return
	}
	overrideServiceImpl.UpdateOverride(&overrideDTO)
	c.JSON(http.StatusOK, true)
}

func SearchOverride(c *gin.Context) {
	service := c.DefaultQuery("service", "")
	application := c.DefaultQuery("application", "")
	//env := c.Param("env")
	serviceVersion := c.DefaultQuery("serviceVersion", "")
	serviceGroup := c.DefaultQuery("serviceGroup", "")
	override := &dto.DynamicConfigDTO{}
	result := make([]dto.DynamicConfigDTO, 0)
	var err error
	if service != "" {
		baseDto := dto.BaseDTO{
			Service:        service,
			ServiceVersion: serviceVersion,
			ServiceGroup:   serviceGroup,
		}
		configDTO := dto.DynamicConfigDTO{
			BaseDTO: baseDto,
			//ServiceVersion: serviceVersion,
			//ServiceGroup:   serviceGroup,
		}
		id := GetIdFromDTO(configDTO.BaseDTO)
		//反射
		override, err = overrideServiceImpl.FindOverride(id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else if application != "" {
		override, err = overrideServiceImpl.FindOverride(application)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Either Service or application is required."})
		return
	}

	if reflect.ValueOf(override).IsNil() {
		result = append(result, *override)
	}

	c.JSON(http.StatusOK, result)
}

func GetIdFromDTO(baseDTO dto.BaseDTO) string {

	if baseDTO.Application != "" {
		return baseDTO.Application
	}
	// id format: "${class}:${version}:${group}"
	var builder strings.Builder
	builder.WriteString(baseDTO.Service)
	builder.WriteString(":")
	builder.WriteString(null2EmptyString(&baseDTO.ServiceVersion))
	builder.WriteString(":")
	builder.WriteString(null2EmptyString(&baseDTO.ServiceGroup))
	return builder.String()
}
func null2EmptyString(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}

func DetailOverride(c *gin.Context) {
	id := c.Param("id")
	//env := c.Param("env")
	override, err := overrideServiceImpl.FindOverride(id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if reflect.ValueOf(override).IsNil() {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Unknown ID!"})
		return
	}
	c.JSON(http.StatusOK, override)
}

func EnableRoute(c *gin.Context) {
	id := c.Param("id")
	//env := c.Param("env")
	overrideServiceImpl.EnableOverride(id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func DeleteOverride(c *gin.Context) {
	id := c.Param("id")
	overrideServiceImpl.DeleteOverride(id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func DisableRoute(c *gin.Context) {
	id := c.Param("id")
	//env := c.Param("env")
	overrideServiceImpl.DisableOverride(id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}
