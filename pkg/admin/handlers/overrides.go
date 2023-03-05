package handlers

import (
	"github.com/apache/dubbo-admin/pkg/admin/model/dto"
	"github.com/apache/dubbo-admin/pkg/admin/util"
	"reflect"

	"errors"
	//"admin/services"
	"github.com/apache/dubbo-admin/pkg/admin/services/impl"
	"github.com/gin-gonic/gin"
	"net/http"
	//"github.com/apache/dubbo-go-admin/service"
	//"github.com/apache/dubbo-go-admin/model/dto"
	//"github.com/apache/dubbo-go-admin/common/exception"
)

var providerServiceImpl impl.ProviderServiceImpl

var overrideServiceImpl impl.OverrideServiceImpl

//var overrideServieImpl services.OverrideService

func CreateOverride(c *gin.Context) {
	var overrideDTO dto.DynamicConfigDTO
	if err := c.BindJSON(&overrideDTO); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	serviceName := overrideDTO.GetService()
	application := overrideDTO.GetApplication()
	if serviceName == "" && application == "" {
		c.AbortWithError(http.StatusBadRequest, errors.New("serviceName and application are Empty!"))
		return
	}

	if application != "" && providerServiceImpl.FindVersionInApplication(application) == "2.6" {

		c.AbortWithError(http.StatusBadRequest, errors.New("dubbo 2.6 does not support application scope dynamic config"))
		return
	}

	overrideServiceImpl.SaveOverride(overrideDTO)
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
	old := overrideServiceImpl.FindOverride(id)
	if reflect.ValueOf(old).IsNil() {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Unknown ID!"})
		return
	}
	overrideServiceImpl.UpdateOverride(overrideDTO)
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
		var util util.Convert
		id := util.GetIdFromDTO(configDTO)
		//反射
		override = overrideServiceImpl.FindOverride(id.(string)).(*dto.DynamicConfigDTO)
	} else if application != "" {
		override = overrideServiceImpl.FindOverride(application).(*dto.DynamicConfigDTO)
	} else {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Either Service or application is required."})
		return
	}

	if override != nil {
		result = append(result, *override)
	}

	c.JSON(http.StatusOK, result)
}

func DetailOverride(c *gin.Context) {
	id := c.Param("id")
	//env := c.Param("env")
	override := overrideServiceImpl.FindOverride(id)
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
