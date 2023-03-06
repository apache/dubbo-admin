package store

type OverrideDTO struct {
	Key           string           `json:"key"`
	Scope         string           `json:"scope"`
	ConfigVersion string           `json:"configVersion"`
	Enabled       bool             `json:"enabled"`
	Configs       []OverrideConfig `json:"configs"`
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

func (o *OverrideDTO) GetConfigs() []OverrideConfig {
	return o.Configs
}

func (o *OverrideDTO) SetConfigs(configs []OverrideConfig) {
	o.Configs = configs
}
