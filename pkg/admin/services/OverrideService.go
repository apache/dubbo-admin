package services

import "github.com/apache/dubbo-admin/pkg/admin/model/dto"

type OverrideService interface {
	SaveOverride(override dto.DynamicConfigDTO)
	UpdateOverride(overrideDTO dto.DynamicConfigDTO)
	DisableOverride(id string)
	FindOverride(id string) interface{}
	EnableOverride(id string)
	DeleteOverride(id string)
}
