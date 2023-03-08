package adapter

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model/dto"
	"github.com/apache/dubbo-admin/pkg/admin/model/store"
)

type DynamicConfigDTO2OverrideDTOAdapter struct {
	store.OverrideDTO
}

func NewDynamicConfigDTO2OverrideDTOAdapter(dynamicConfigDTO *dto.DynamicConfigDTO) *DynamicConfigDTO2OverrideDTOAdapter {
	adapter := &DynamicConfigDTO2OverrideDTOAdapter{}

	if dynamicConfigDTO.Application != "" {
		adapter.Scope = constant.ApplicationKey
		adapter.Key = dynamicConfigDTO.Application
	} else {
		adapter.Scope = constant.ServiceKey
		adapter.Key = dynamicConfigDTO.Service
	}

	adapter.ConfigVersion = dynamicConfigDTO.ConfigVersion
	adapter.Configs = dynamicConfigDTO.Configs
	return adapter
}
