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

package console

import (
	"fmt"
	"net/url"

	set "github.com/duke-git/lancet/v2/datastructure/set"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/config"
	"github.com/apache/dubbo-admin/pkg/config/console/auth"
)

type GinRunningMode string

var supportedGinRunningMode = set.New(DebugMode, ReleaseMode, TestMode)

const (
	DebugMode   GinRunningMode = gin.DebugMode
	ReleaseMode GinRunningMode = gin.ReleaseMode
	TestMode    GinRunningMode = gin.TestMode
)

type Config struct {
	config.BaseConfig
	GinMode          GinRunningMode          `json:"ginMode" yaml:"ginMode"`
	Port             int                     `json:"port" yaml:"port"`
	MetricDashboards *GrafanaDashboardConfig `json:"metricDashboards" yaml:"metricDashboards"`
	TraceDashboards  *GrafanaDashboardConfig `json:"traceDashboards" yaml:"traceDashboards"`
	Auth             *auth.Config            `json:"auth" yaml:"auth"`
}

func (c *Config) Sanitize() {
	if c.Auth != nil {
		c.Auth.Sanitize()
	}
}

func (c *Config) Validate() error {
	if !supportedGinRunningMode.Contain(c.GinMode) {
		return bizerror.New(bizerror.ConfigError, fmt.Sprintf("invalid gin mode: %s", c.GinMode))
	}
	if c.Port < 1 || c.Port > 65535 {
		return bizerror.New(bizerror.ConfigError, "port of console must between 1 to 65535")
	}
	if c.Auth == nil {
		return bizerror.New(bizerror.ConfigError, "auth config is needed, but found empty")
	}
	if c.MetricDashboards != nil {
		if err := c.MetricDashboards.Validate(); err != nil {
			return err
		}
	}
	if c.TraceDashboards != nil {
		if err := c.TraceDashboards.Validate(); err != nil {
			return err
		}
	}
	if err := c.Auth.Validate(); err != nil {
		return err
	}
	// Release deployments with external providers must reject the legacy default session secret and other short cookie-signing keys.
	if c.GinMode == ReleaseMode && len(c.Auth.Providers) > 0 && len([]byte(c.Auth.SessionSecret)) < 32 {
		return bizerror.New(bizerror.ConfigError, "auth sessionSecret must contain at least 32 bytes when providers are enabled in release mode")
	}
	return nil
}

type GrafanaDashboardConfig struct {
	config.BaseConfig
	// application level metrics panel
	Application string `json:"application" yaml:"application"`
	// instance level metrics panel
	Instance string `json:"instance" yaml:"instance"`
	// service level metrics panel
	Service string `json:"service" yaml:"service"`

	AppDashboardURL      *url.URL `json:"-" yaml:"-"`
	InstanceDashboardURL *url.URL `json:"-" yaml:"-"`
	ServiceDashboardURL  *url.URL `json:"-" yaml:"-"`
}

func (g *GrafanaDashboardConfig) Validate() error {
	if strutil.IsNotBlank(g.Application) {
		dashboardURL, err := url.Parse(g.Application)
		if err != nil {
			return bizerror.Wrap(err, bizerror.ConfigError,
				fmt.Sprintf("invalid application dashboard url: %s", g.Application))
		}
		g.AppDashboardURL = dashboardURL
	}
	if strutil.IsNotBlank(g.Instance) {
		dashboardURL, err := url.Parse(g.Instance)
		if err != nil {
			return bizerror.Wrap(err, bizerror.ConfigError,
				fmt.Sprintf("invalid instance dashboard url: %s", g.Instance))
		}
		g.InstanceDashboardURL = dashboardURL
	}
	if strutil.IsNotBlank(g.Service) {
		dashboardURL, err := url.Parse(g.Service)
		if err != nil {
			return bizerror.Wrap(err, bizerror.ConfigError,
				fmt.Sprintf("invalid service dashboard url: %s", g.Service))
		}
		g.ServiceDashboardURL = dashboardURL
	}
	return nil
}

func DefaultConsoleConfig() *Config {
	return &Config{
		GinMode: ReleaseMode,
		Port:    8888,
		Auth: &auth.Config{
			Methods:        []string{auth.MethodPassword},
			User:           "admin",
			Password:       "admin",
			ExpirationTime: 3600,
			SessionSecret:  auth.DefaultSessionSecret,
		},
	}
}
