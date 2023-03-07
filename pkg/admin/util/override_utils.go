package util

import (
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model/dto"
	"github.com/apache/dubbo-admin/pkg/admin/model/store"
)

func CreateFromOverride(overrideDTO dto.OverrideDTO) *dto.DynamicConfigDTO {
	dynamicConfigDTO := &dto.DynamicConfigDTO{}
	dynamicConfigDTO.SetConfigVersion(overrideDTO.GetConfigVersion())
	configs := []store.OverrideConfig{}
	for _, overrideConfig := range overrideDTO.GetConfigs() {
		if overrideConfig.GetType() == "" {
			configs = append(configs, overrideConfig)
		}
	}

	if len(configs) == 0 {
		return nil
	}

	dynamicConfigDTO.SetConfigs(configs)

	if overrideDTO.GetScope() == constant.ApplicationKey {
		dynamicConfigDTO.SetApplication(overrideDTO.GetKey())
	} else {
		dynamicConfigDTO.SetService(overrideDTO.GetKey())
	}

	dynamicConfigDTO.SetEnabled(overrideDTO.IsEnabled())
	return dynamicConfigDTO
}
