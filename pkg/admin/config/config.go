/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/apache/dubbo-admin/pkg/admin/constant"
	_ "github.com/apache/dubbo-admin/pkg/admin/imports"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/logger"
	"gopkg.in/yaml.v2"
)

const conf = "./conf/dubboadmin.yml"

type Config struct {
	Admin      Admin      `yaml:"admin"`
	Prometheus Prometheus `yaml:"prometheus"`
}

type Prometheus struct {
	Ip   string `json:"ip"`
	Port string `json:"port"`
}

type Admin struct {
	ConfigCenter   string        `yaml:"config-center"`
	MetadataReport AddressConfig `yaml:"metadata-report"`
	Registry       AddressConfig `yaml:"registry"`
	MysqlDsn       string        `yaml:"mysql-dsn"`
}

var (
	ConfigCenter         config_center.DynamicConfiguration
	RegistryCenter       registry.Registry
	MetadataReportCenter report.MetadataReport

	DataBase *gorm.DB // for service mock

	Group string
)

var (
	PrometheusIp   string
	PrometheusPort string
)

func LoadConfig() {
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
	metadataReportAddress := config.Admin.MetadataReport.Address

	loadDatabaseConfig(config.Admin.MysqlDsn)

	PrometheusIp = config.Prometheus.Ip
	PrometheusPort = config.Prometheus.Port
	if PrometheusIp == "" {
		PrometheusIp = "127.0.0.1"
	}
	if PrometheusPort == "" {
		PrometheusPort = "9090"
	}

	c, addrUrl := getValidAddressConfig(address, registryAddress)
	ConfigCenter = newConfigCenter(c, addrUrl)
	properties, err := ConfigCenter.GetProperties(constant.DubboPropertyKey)
	if err != nil {
		logger.Info("No configuration found in config center.")
	}

	if len(properties) > 0 {
		logger.Info("Loaded remote configuration from config center:\n %s", properties)
		for _, property := range strings.Split(properties, "\n") {
			if strings.HasPrefix(property, constant.RegistryAddressKey) {
				registryAddress = strings.Split(property, "=")[1]
			}
			if strings.HasPrefix(property, constant.MetadataReportAddressKey) {
				metadataReportAddress = strings.Split(property, "=")[1]
			}
		}
	}

	if len(registryAddress) > 0 {
		logger.Info("Valid registry address is %s", registryAddress)
		c := newAddressConfig(registryAddress)
		addrUrl, err := c.toURL()
		if err != nil {
			panic(err)
		}
		RegistryCenter, err = extension.GetRegistry(c.getProtocol(), addrUrl)
		if err != nil {
			panic(err)
		}
	}
	if len(metadataReportAddress) > 0 {
		logger.Info("Valid meta center address is %s", metadataReportAddress)
		c := newAddressConfig(metadataReportAddress)
		addrUrl, err := c.toURL()
		if err != nil {
			panic(err)
		}
		factory := extension.GetMetadataReportFactory(c.getProtocol())
		MetadataReportCenter = factory.CreateMetadataReport(addrUrl)
	}
}

func getValidAddressConfig(address string, registryAddress string) (AddressConfig, *common.URL) {
	if len(address) <= 0 && len(registryAddress) <= 0 {
		panic("Must at least specify `admin.config-center.address` or `admin.registry.address`!")
	}

	var c AddressConfig
	if len(address) > 0 {
		logger.Info("Specified config center address is %s", address)
		c = newAddressConfig(address)
	} else {
		logger.Info("Using registry address as default config center address")
		c = newAddressConfig(registryAddress)
	}

	configUrl, err := c.toURL()
	if err != nil {
		panic(err)
	}
	return c, configUrl
}

func newConfigCenter(c AddressConfig, url *common.URL) config_center.DynamicConfiguration {
	factory, err := extension.GetConfigCenterFactory(c.getProtocol())
	if err != nil {
		logger.Info(err.Error())
		panic(err)
	}

	configCenter, err := factory.GetDynamicConfiguration(url)
	if err != nil {
		logger.Info("Failed to init config center, error msg is %s.", err.Error())
		panic(err)
	}
	return configCenter
}

func newAddressConfig(address string) AddressConfig {
	config := AddressConfig{}
	config.Address = address
	var err error
	config.url, err = url.Parse(address)
	if err != nil {
		panic(err)
	}
	return config
}

// load database for mock rule storage, if dsn is empty, use sqlite in memory
func loadDatabaseConfig(dsn string) {
	var db *gorm.DB
	var err error
	if dsn == "" {
		db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	} else {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	}
	if err != nil {
		panic(err)
	} else {
		DataBase = db
		// init table
		initErr := DataBase.AutoMigrate(&model.MockRuleEntity{})
		if initErr != nil {
			panic(initErr)
		}
	}
}
