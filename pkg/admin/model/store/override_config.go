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

func (c *OverrideConfig) GetSide() string {
	return c.Side
}

func (c *OverrideConfig) SetSide(side string) {
	c.Side = side
}

func (c *OverrideConfig) GetAddresses() []string {
	return c.Addresses
}

func (c *OverrideConfig) SetAddresses(addresses []string) {
	c.Addresses = addresses
}

func (c *OverrideConfig) GetProviderAddresses() []string {
	return c.ProviderAddresses
}

func (c *OverrideConfig) SetProviderAddresses(providerAddresses []string) {
	c.ProviderAddresses = providerAddresses
}

func (c *OverrideConfig) GetParameters() map[string]interface{} {
	return c.Parameters
}

func (c *OverrideConfig) SetParameters(parameters map[string]interface{}) {
	c.Parameters = parameters
}

func (c *OverrideConfig) GetApplications() []string {
	return c.Applications
}

func (c *OverrideConfig) SetApplications(applications []string) {
	c.Applications = applications
}

func (c *OverrideConfig) GetServices() []string {
	return c.Services
}

func (c *OverrideConfig) SetServices(services []string) {
	c.Services = services
}

func (c *OverrideConfig) GetType() string {
	return c.Type
}

func (c *OverrideConfig) SetType(typ string) {
	c.Type = typ
}

func (c *OverrideConfig) IsEnabled() bool {
	return c.Enabled
}

func (c *OverrideConfig) SetEnabled(enabled bool) {
	c.Enabled = enabled
}
