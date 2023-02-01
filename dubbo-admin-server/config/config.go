package config

import (
	"admin/constant"
	_ "admin/imports"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strings"
)

const conf = "../conf/dubboadmin.yml"

type Config struct {
	Admin Admin `yaml:"admin"`
}

type Admin struct {
	ConfigCenter   string        `yaml:"config-center"`
	MetadataReport AddressConfig `yaml:"metadata-report"`
	Registry       AddressConfig `yaml:"registry"`
}

var (
	ConfigCenter config_center.DynamicConfiguration
)

func LoadConfig() Config {
	path, err := filepath.Abs(conf)
	if err != nil {
		path = filepath.Clean(conf)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var config Config
	yaml.Unmarshal(content, &config)
	address := config.Admin.ConfigCenter
	registryAddress := config.Admin.Registry.Address
	//metadataReportAddress := config.Admin.MetadataReport.address
	if len(address) > 0 {
		c := AddressConfig{}
		c.Address = address
		factory, err := extension.GetConfigCenterFactory(c.getProtocol())
		if err != nil {
			panic(err)
		}
		url, err := c.toURL()
		if err != nil {
			panic(err)
		}
		ConfigCenter, err = factory.GetDynamicConfiguration(url)
		properties, err := ConfigCenter.GetProperties(constant.DubboPropertyKey)
		if err != nil {
			panic(err)
		}
		if len(properties) > 0 {
			for _, property := range strings.Split(properties, "\n") {
				if strings.HasPrefix(property, constant.RegistryAddressKey) {
					registryAddress = strings.Split(property, "=")[1]
				}
				if strings.HasPrefix(property, constant.MetadataReportAddressKey) {
					//metadataReportAddress := strings.Split(property, "=")[1]
				}
			}
		}

	}
	if ConfigCenter == nil {
		if len(registryAddress) == 0 {
			panic("registry address can not be empty")
		}

	}
	return config
}
