package services

import "admin/pkg/model"

type ProviderService interface {
	FindServices() []string
	FindService(string, string) []*model.Provider
}
