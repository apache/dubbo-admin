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

package admin

import (
	"net/url"
	"strings"

	"github.com/apache/dubbo-admin/pkg/admin/cache/registry"

	"github.com/apache/dubbo-admin/pkg/admin/providers/mock"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/apache/dubbo-admin/pkg/core/logger"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/config/admin"
	core_runtime "github.com/apache/dubbo-admin/pkg/core/runtime"

	_ "github.com/apache/dubbo-admin/pkg/admin/cache/registry/kube"
	_ "github.com/apache/dubbo-admin/pkg/admin/cache/registry/universal"
)

func RegisterDatabase(rt core_runtime.Runtime) error {
	dsn := rt.Config().Admin.MysqlDSN
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
		config.DataBase = db
		// init table
		initErr := config.DataBase.AutoMigrate(&model.MockRuleEntity{})
		if initErr != nil {
			panic(initErr)
		}
	}
	return nil
}

func RegisterOther(rt core_runtime.Runtime) error {
	config.AdminPort = rt.Config().Admin.AdminPort
	config.PrometheusIp = rt.Config().Admin.Prometheus.Ip
	config.PrometheusPort = rt.Config().Admin.Prometheus.Port
	config.PrometheusMonitorPort = rt.Config().Admin.Prometheus.MonitorPort
	address := rt.Config().Admin.ConfigCenter
	registryAddress := rt.Config().Admin.Registry.Address
	metadataReportAddress := rt.Config().Admin.MetadataReport.Address
	c, addrUrl := getValidAddressConfig(address, registryAddress)
	configCenter := newConfigCenter(c, addrUrl)
	config.Governance = config.NewGovernanceConfig(configCenter, c.GetProtocol())
	properties, err := configCenter.GetProperties(constant.DubboPropertyKey)
	if err != nil {
		logger.Info("No configuration found in config center.")
	}
	if len(properties) > 0 {
		logger.Infof("Loaded remote configuration from config center:\n %s", properties)
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
		logger.Infof("Valid registry address is %s", registryAddress)
		c := newAddressConfig(registryAddress)
		addrUrl, err := c.ToURL()
		if err != nil {
			panic(err)
		}

		config.RegistryCenter, err = extension.GetRegistry(c.GetProtocol(), addrUrl)
		if err != nil {
			panic(err)
		}
		config.AdminRegistry, err = registry.Registry(c.GetProtocol(), addrUrl)
		if err != nil {
			panic(err)
		}
	}
	if len(metadataReportAddress) > 0 {
		logger.Infof("Valid meta center address is %s", metadataReportAddress)
		c := newAddressConfig(metadataReportAddress)
		addrUrl, err := c.ToURL()
		if err != nil {
			panic(err)
		}
		factory := extension.GetMetadataReportFactory(c.GetProtocol())
		config.MetadataReportCenter = factory.CreateMetadataReport(addrUrl)
	}

	// start go routines to subscribe to registries
	services.StartSubscribe(config.AdminRegistry)
	defer func() {
		services.DestroySubscribe(config.AdminRegistry)
	}()

	// start mock cp-server
	go mock.RunMockServiceServer(rt.Config().Admin, rt.Config().Dubbo)

	return nil
}

func getValidAddressConfig(address string, registryAddress string) (admin.AddressConfig, *common.URL) {
	if len(address) <= 0 && len(registryAddress) <= 0 {
		panic("Must at least specify `admin.config-center.address` or `admin.registry.address`!")
	}

	var c admin.AddressConfig
	if len(address) > 0 {
		logger.Infof("Specified config center address is %s", address)
		c = newAddressConfig(address)
	} else {
		logger.Info("Using registry address as default config center address")
		c = newAddressConfig(registryAddress)
	}

	configUrl, err := c.ToURL()
	if err != nil {
		panic(err)
	}
	return c, configUrl
}

func newAddressConfig(address string) admin.AddressConfig {
	config := admin.AddressConfig{}
	config.Address = address
	var err error
	config.Url, err = url.Parse(address)
	if err != nil {
		panic(err)
	}
	return config
}

func newConfigCenter(c admin.AddressConfig, url *common.URL) config_center.DynamicConfiguration {
	factory, err := extension.GetConfigCenterFactory(c.GetProtocol())
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
