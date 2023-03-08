package store

type OverrideDTO struct {
	Key           string           `json:"key"`
	Scope         string           `json:"scope"`
	ConfigVersion string           `json:"configVersion"`
	Enabled       bool             `json:"enabled"`
	Configs       []OverrideConfig `json:"configs"`
}
