package services

import (
	"admin/pkg/cache"
	"admin/pkg/constant"
	"admin/pkg/model"
	"admin/pkg/util"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"sync"
)

type ProviderServiceImpl struct{}

func (p *ProviderServiceImpl) FindServices() []string {
	servicesMap, ok := cache.InterfaceRegistryCache.Load(constant.ProvidersCategory)
	var services []string
	if !ok {
		return services
	}
	servicesMap.(*sync.Map).Range(func(key, v interface{}) bool {
		services = append(services, key.(string))
		return true
	})
	return services
}

func (p *ProviderServiceImpl) findByService(serviceName string) []*model.Provider {
	var providers []*model.Provider
	addProvider := func(serviceMap any) {
		for id, url := range serviceMap.(map[string]*common.URL) {
			provider := util.URL2Provider(id, url)
			if provider != nil {
				providers = append(providers, provider)
			}
		}
	}
	services, ok := cache.InterfaceRegistryCache.Load(constant.ProvidersCategory)
	if !ok {
		return providers
	}
	servicesMap, ok := services.(*sync.Map)
	if !ok {
		// servicesMap type error
		logger.Error("servicesMap type not *sync.Map")
		return providers
	}
	if serviceName == constant.AnyValue {
		servicesMap.Range(func(key, value any) bool {
			addProvider(value)
			return true
		})
	}
	serviceMap, ok := servicesMap.Load(serviceName)
	if !ok {
		return providers
	}
	addProvider(serviceMap)
	return providers
}

func (p *ProviderServiceImpl) FindService(pattern string, filter string) []*model.Provider {
	if pattern == constant.Service {
		return p.findByService(filter)
	}
	return nil
}
