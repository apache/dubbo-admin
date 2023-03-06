package dto

type BaseDTO struct {
	Application    string `json:"application"`
	Service        string `json:"service"`
	ID             string `json:"id"`
	ServiceVersion string `json:"serviceVersion"`
	ServiceGroup   string `json:"serviceGroup"`
}

func (b *BaseDTO) GetApplication() string {
	return b.Application
}

func (b *BaseDTO) SetApplication(application string) {
	b.Application = application
}

func (b *BaseDTO) GetService() string {
	return b.Service
}

func (b *BaseDTO) SetService(service string) {
	b.Service = service
}

func (b *BaseDTO) GetID() string {
	return b.ID
}

func (b *BaseDTO) SetID(id string) {
	b.ID = id
}

func (b *BaseDTO) GetServiceVersion() string {
	return b.ServiceVersion
}

func (b *BaseDTO) SetServiceVersion(serviceVersion string) {
	b.ServiceVersion = serviceVersion
}

func (b *BaseDTO) GetServiceGroup() string {
	return b.ServiceGroup
}

func (b *BaseDTO) SetServiceGroup(serviceGroup string) {
	b.ServiceGroup = serviceGroup
}
