package dto

type BaseDTO struct {
	Application    string `json:"application"`
	Service        string `json:"service"`
	ID             string `json:"id"`
	ServiceVersion string `json:"serviceVersion"`
	ServiceGroup   string `json:"serviceGroup"`
}
