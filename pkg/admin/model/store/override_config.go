package store

type OverrideConfig struct {
	Side              string                 `json:"side"`
	Addresses         []string               `json:"addresses"`
	ProviderAddresses []string               `json:"providerAddresses"`
	Parameters        map[string]interface{} `json:"parameters"`
	Applications      []string               `json:"applications"`
	Services          []string               `json:"services"`
	Type              string                 `json:"type"`
	Enabled           bool                   `json:"enabled"`
}
