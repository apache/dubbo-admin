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

package dubbo_cp

import (
	"time"

	dubbogo "dubbo.apache.org/dubbo-go/v3/config"
	"github.com/apache/dubbo-admin/pkg/config"
	"github.com/apache/dubbo-admin/pkg/config/option"
	"github.com/pkg/errors"

	"github.com/apache/dubbo-admin/pkg/config/admin"
	"github.com/apache/dubbo-admin/pkg/config/kube"
	"github.com/apache/dubbo-admin/pkg/config/security"
	"github.com/apache/dubbo-admin/pkg/config/server"
)

type Config struct {
	Admin      admin.Admin             `yaml:"admin"`
	GrpcServer server.ServerConfig     `yaml:"grpc-cp-server"`
	Security   security.SecurityConfig `yaml:"security"`
	KubeConfig kube.KubeConfig         `yaml:"kube-config"`
	Dubbo      dubbogo.RootConfig      `yaml:"dubbo"`
	Options    option.Options          `yaml:"options"`
}

func (c *Config) Sanitize() {
	c.Security.Sanitize()
	c.Admin.Sanitize()
	c.GrpcServer.Sanitize()
	c.KubeConfig.Sanitize()
	c.Options.Sanitize()
}

func (c *Config) Validate() error {
	err := c.Security.Validate()
	if err != nil {
		return errors.Wrap(err, "SecurityConfig validation failed")
	}
	err = c.Admin.Validate()
	if err != nil {
		return errors.Wrap(err, "Admin validation failed")
	}
	err = c.GrpcServer.Validate()
	if err != nil {
		return errors.Wrap(err, "ServerConfig validation failed")
	}
	err = c.KubeConfig.Validate()
	if err != nil {
		return errors.Wrap(err, "KubeConfig validation failed")
	}
	err = c.Options.Validate()
	if err != nil {
		return errors.Wrap(err, "options validation failed")
	}
	return nil
}

var DefaultConfig = func() Config {
	return Config{
		Admin: admin.Admin{
			AdminPort:    38080,
			ConfigCenter: "zookeeper://127.0.0.1:2181",
			MetadataReport: admin.AddressConfig{
				Address: "zookeeper://127.0.0.1:2181",
			},
			Registry: admin.AddressConfig{
				Address: "zookeeper://127.0.0.1:2181",
			},
			Prometheus: admin.Prometheus{
				Ip:          "127.0.0.1",
				Port:        "9090",
				MonitorPort: "22222",
			},
			// MysqlDSN: "root:password@tcp(127.0.0.1:3306)/dubbo-admin?charset=utf8&parseTime=true",
		},
		GrpcServer: server.ServerConfig{
			PlainServerPort:  30060,
			SecureServerPort: 30062,
			DebugPort:        30070,
		},
		Security: security.SecurityConfig{
			CaValidity:           30 * 24 * 60 * 60 * 1000,
			CertValidity:         1 * 60 * 60 * 1000,
			IsTrustAnyone:        false,
			EnableOIDCCheck:      true,
			ResourcelockIdentity: config.GetStringEnv("POD_NAME", config.GetDefaultResourcelockIdentity()),
			WebhookPort:          30080,
			WebhookAllowOnErr:    true,
		},
		KubeConfig: kube.KubeConfig{
			Namespace:             "dubbo-system",
			ServiceName:           "dubbo-cp",
			InPodEnv:              false,
			IsKubernetesConnected: false,
			RestConfigQps:         50,
			RestConfigBurst:       100,
			KubeFileConfig:        "",
			DomainSuffix:          "cluster.local",
		},
		Dubbo: dubbogo.RootConfig{},
		Options: option.Options{
			DebounceAfter:   100 * time.Millisecond,
			DebounceMax:     10 * time.Second,
			EnableDebounce:  true,
			SendTimeout:     5 * time.Second,
			DdsBlockMaxTime: 15 * time.Second,
		},
	}
}
