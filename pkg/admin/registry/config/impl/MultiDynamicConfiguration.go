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

func (m *MultiDynamicConfiguration) Init() {
	if reflect.ValueOf(m.url).IsNil() {
		panic("server url is null, cannot init")
	}
	var dynamicConfigFactory config_center.DynamicConfigurationFactory
	//TODO  多态的实现
	//dynamicConfigurationFactory := config_center.DynamicConfigurationFactory(m.url)
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

//func (m *MultiDynamicConfiguration) GetConfig(key string) string {
//	return m.GetConfigWithGroup(m.group, key)
//}

func (m *MultiDynamicConfiguration) DeleteConfig(key string) bool {
	return m.DeleteConfigWithGroup(m.group, key)
}

// TODO   getConfig该调用哪个
//func (m *MultiDynamicConfiguration) GetConfigWithGroup(group, key string) string {
//	if key == "" {
//		panic("key cannot be empty")
//	}
//	return m.dynamicConfiguration.GetConfig(key, group)
//}

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
