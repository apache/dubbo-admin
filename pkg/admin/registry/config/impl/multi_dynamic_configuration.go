package impl

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"reflect"
)

type MultiDynamicConfiguration struct {
	url                  common.URL
	dynamicConfiguration config_center.DynamicConfiguration
	group                string
}

type DynamicConfig interface {
	config_center.DynamicConfiguration
	GetConfig(string, string) string
}

type DynamicConfigImpl struct {
	a config_center.DynamicConfiguration
}

//func (a *DynamicConfigImpl) GetConfig(key, Group string) string {
//	//return a.(key, group, this.getDefaultTimeout())
//}

func (m *MultiDynamicConfiguration) Init() {
	if reflect.ValueOf(m.url).IsNil() {
		panic("server url is null, cannot init")
	}
	var dynamicConfigFactory config_center.DynamicConfigurationFactory
	//TODO  多态的实现  存在问题

	//dynamicConfigurationFactory,err := dynamicConfigFactory.GetDynamicConfiguration(&m.url)
	//if err!=nil{
	//	log.Fatal("dynamicConfigurationFactory get error",err)
	//}
	//dynamicconfig.GetDynamicConfigurationFactory(m.url.Scheme)

	m.dynamicConfiguration, _ = dynamicConfigFactory.GetDynamicConfiguration(&m.url)
	m.group = m.url.GetParam(constant.GroupKey, constant.Dubbo)
}

func (m *MultiDynamicConfiguration) SetUrl(url common.URL) {
	m.url = url
}

func (m *MultiDynamicConfiguration) GetUrl() common.URL {
	return m.url
}

func (m *MultiDynamicConfiguration) SetConfig(key, value string) bool {
	return m.SetConfigWithGroup(m.group, key, value)
}
func (m *MultiDynamicConfiguration) SetConfigWithGroup(group, key, value string) bool {
	if key == "" || value == "" {
		panic("key or value cannot be empty")
	}
	//更改逻辑  原逻辑是返回bool  TODO 按照实际更改
	if m.dynamicConfiguration.PublishConfig(key, group, value) == nil {
		return true
	}
	return false
}

func (m *MultiDynamicConfiguration) GetConfig(key string) string {
	return m.GetConfigWithGroup(m.group, key)
}

func (m *MultiDynamicConfiguration) DeleteConfig(key string) bool {
	return m.DeleteConfigWithGroup(m.group, key)
}

func (m *MultiDynamicConfiguration) GetConfigWithGroup(group, key string) string {
	if key == "" {
		panic("key cannot be empty")
	}
	properties, err := m.dynamicConfiguration.GetProperties(key)
	if err != nil {
		return ""
	}
	return properties
}

func (m *MultiDynamicConfiguration) DeleteConfigWithGroup(group, key string) bool {
	if key == "" {
		panic("key cannot be empty")
	}
	if m.dynamicConfiguration.RemoveConfig(key, group) == nil {
		return true
	}
	return false
}

func (m *MultiDynamicConfiguration) GetPath(key string) string {
	return ""
}

func (m *MultiDynamicConfiguration) GetPathWithGroup(group, key string) string {
	return ""
}

//func init() {
//	var dynamicConfiguration config.GovernanceConfiguration = nil
//
//	if conf.ConfigCenter != nil {
//		configCenterUrl := formUrl(conf.Config.Admin.ConfigCenter, configCenterGroup, configCenterGroupNameSpace, username, password)
//		logger.Info("Admin using config center: " + configCenterUrl)
//
//		dynamicConfiguration
//		dynamicConfiguration = Centerconfig.config.DynamicConfigurationFactory()
//		dynamicConfiguration = ExtensionLoader.GetDefaultExtension(GovernanceConfiguration{})
//		dynamicConfiguration.SetUrl(configCenterUrl)
//		dynamicConfiguration.Init()
//		config := dynamicConfiguration.GetConfig(Constants.DUBBO_PROPERTY)
//
//		if config != "" {
//			lines := strings.Split(config, "\n")
//			for _, s := range lines {
//				if strings.HasPrefix(s, Constants.REGISTRY_ADDRESS) {
//					registryAddress := removerConfigKey(s)
//					registryUrl = formUrl(registryAddress, registryGroup, registryNameSpace, username, password)
//				} else if strings.HasPrefix(s, Constants.METADATA_ADDRESS) {
//					metadataUrl = formUrl(removerConfigKey(s), metadataGroup, metadataGroupNameSpace, username, password)
//				}
//				logger.Info("Registry address found from config center: " + registryUrl)
//				logger.Info("Metadata address found from config center: " + registryUrl)
//			}
//		}
//	}
//
//	if dynamicConfiguration == nil {
//		if registryAddress != "" {
//			registryUrl = formUrl(registryAddress, registryGroup, registryNameSpace, username, password)
//			dynamicConfiguration = ExtensionLoader.GetDefaultExtension(GovernanceConfiguration{})
//			dynamicConfiguration.SetUrl(registryUrl)
//			dynamicConfiguration.Init()
//			logger.Warn("you are using dubbo.registry.address, which is not recommend, please refer to: https://github.com/apache/dubbo-admin/wiki/Dubbo-Admin-configuration")
//		} else {
//			panic(ConfigurationException{"Either config center or registry address is needed, please refer to https://github.com/apache/dubbo-admin/wiki/Dubbo-Admin-configuration"})
//		}
//	}
//
//	return dynamicConfiguration
//}

func formUrl(address string, group string, namespace string, username string, password string) string {
	// TODO: implement formUrl function
	return ""
}

func removerConfigKey(config string) string {
	// TODO: implement removerConfigKey function
	return ""
}
