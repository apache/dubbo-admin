package util

import (
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model/dto"
	"github.com/apache/dubbo-admin/pkg/admin/model/store"
)

func CreateFromOverride(overrideDTO store.OverrideDTO) *dto.DynamicConfigDTO {
	dynamicConfigDTO := &dto.DynamicConfigDTO{}
	dynamicConfigDTO.ConfigVersion = overrideDTO.ConfigVersion
	configs := []store.OverrideConfig{}
	for _, overrideConfig := range overrideDTO.Configs {
		if overrideConfig.Type == "" {
			configs = append(configs, overrideConfig)
		}
	}

	if len(configs) == 0 {
		return nil
	}

	dynamicConfigDTO.Configs = configs

	if overrideDTO.Scope == constant.ApplicationKey {
		dynamicConfigDTO.Application = overrideDTO.Key
	} else {
		dynamicConfigDTO.Service = overrideDTO.Key
	}

	dynamicConfigDTO.Enabled = overrideDTO.Enabled
	return dynamicConfigDTO
}
