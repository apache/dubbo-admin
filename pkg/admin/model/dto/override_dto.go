package dto

import "github.com/apache/dubbo-admin/pkg/admin/model/store"

type OverrideDTO struct {
	Key           string
	Scope         string
	ConfigVersion string
	Enabled       bool
	Configs       []store.OverrideConfig
}

func (o *OverrideDTO) GetKey() string {
	return o.Key
}

func (o *OverrideDTO) SetKey(key string) {
	o.Key = key
}

func (o *OverrideDTO) GetScope() string {
	return o.Scope
}

func (o *OverrideDTO) SetScope(scope string) {
	o.Scope = scope
}

func (o *OverrideDTO) GetConfigVersion() string {
	return o.ConfigVersion
}

func (o *OverrideDTO) SetConfigVersion(configVersion string) {
	o.ConfigVersion = configVersion
}

func (o *OverrideDTO) IsEnabled() bool {
	return o.Enabled
}

func (o *OverrideDTO) SetEnabled(enabled bool) {
	o.Enabled = enabled
}

func (o *OverrideDTO) GetConfigs() []store.OverrideConfig {
	return o.Configs
}

//func (o *OverrideDTO) SetConfigs(configs []*OverrideConfig) {
//	o.Configs = configs
//}
