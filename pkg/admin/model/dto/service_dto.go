package dto

import (
	"github.com/apache/dubbo-admin/pkg/admin/model"
)

type ServiceDTO struct {
	Service        string
	AppName        string
	Group          string
	Version        string
	RegistrySource model.RegistrySource
}
