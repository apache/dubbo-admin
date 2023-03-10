package util

import (
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
)

func GetIdFromDto(baseDto model.BaseDto) string {
	if baseDto.Application != "" {
		return baseDto.Application
	}
	// id format: "${class}:${version}:${group}"
	return baseDto.Service + constant.Colon + baseDto.ServiceVersion + constant.Colon + baseDto.ServiceGroup
}
