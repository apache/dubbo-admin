package services

import "github.com/apache/dubbo-admin/pkg/admin/registry/config"

//func a() {
//	extension.S
//}

// var logger *Logger = NewLogger()
// var registry Registry
//var dynamicConfiguration GovernanceConfiguration

//var metaDataCollector MetaDataCollector
//var interfaceRegistryCache InterfaceRegistryCache = NewInterfaceRegistryCache()

type AbstractService struct {
	dynamicConfiguration config.GovernanceConfiguration
}

//func (service *AbstractService) GetInterfaceRegistryCache() sync.Map {
//	return interfaceRegistryCache.GetRegistryCache()
//}
