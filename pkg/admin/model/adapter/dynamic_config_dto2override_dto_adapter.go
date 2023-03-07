package adapter

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model/dto"
	"github.com/apache/dubbo-admin/pkg/admin/model/store"
)

type DynamicConfigDTO2OverrideDTOAdapter struct {
	store.OverrideDTO
}

func NewDynamicConfigDTO2OverrideDTOAdapter(dynamicConfigDTO dto.DynamicConfigDTO) (*DynamicConfigDTO2OverrideDTOAdapter, error) {
	adapter := &DynamicConfigDTO2OverrideDTOAdapter{}

	if dynamicConfigDTO.Application != "" {
		adapter.SetScope(constant.ApplicationKey)
		adapter.SetKey(dynamicConfigDTO.Application)
	} else {
		adapter.SetScope(constant.ServiceKey)
		adapter.SetKey(dynamicConfigDTO.Service)
	}

	adapter.SetConfigVersion(dynamicConfigDTO.ConfigVersion)
	adapter.SetConfigs(dynamicConfigDTO.Configs)
	return adapter, nil
}
