package services

import "github.com/apache/dubbo-admin/pkg/admin/model/dto"

type OverrideService interface {
	SaveOverride(override *dto.DynamicConfigDTO) error
	UpdateOverride(overrideDTO *dto.DynamicConfigDTO) error
	DisableOverride(id string) error
	FindOverride(id string) (*dto.DynamicConfigDTO, error)
	EnableOverride(id string) error
	DeleteOverride(id string) error
}
