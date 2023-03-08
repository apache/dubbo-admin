package dto

import (
	"github.com/apache/dubbo-admin/pkg/admin/model/store"
	_ "github.com/apache/dubbo-admin/pkg/admin/model/store"
)

type DynamicConfigDTO struct {
	BaseDTO
	ConfigVersion string                 `json:"configVersion"`
	Enabled       bool                   `json:"enabled"`
	Configs       []store.OverrideConfig `json:"configs"`
}
