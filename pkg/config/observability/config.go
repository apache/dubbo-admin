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

package observability

import (
	"fmt"
	"net/url"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/config"
)

type Config struct {
	config.BaseConfig
	// Grafana is the url of grafana
	Grafana string `json:"grafana" yaml:"grafana"`
	// Prometheus is the url of prometheus
	Prometheus string `json:"prometheus" yaml:"prometheus"`
	// Logs configures the log query provider.
	Logs *LogsConfig `json:"logs,omitempty" yaml:"logs,omitempty"`

	GrafanaBaseURL    *url.URL `json:"-" yaml:"-"`
	PrometheusBaseURL *url.URL `json:"-" yaml:"-"`
}

func (c *Config) Validate() error {
	if strutil.IsNotBlank(c.Prometheus) {
		prometheusBaseURL, err := url.Parse(c.Prometheus)
		if err != nil {
			return bizerror.Wrap(err, bizerror.ConfigError,
				fmt.Sprintf("invalid prometheus url: %s", c.Prometheus))
		}
		c.PrometheusBaseURL = prometheusBaseURL
	}
	if strutil.IsNotBlank(c.Grafana) {
		grafanaBaseURL, err := url.Parse(c.Grafana)
		if err != nil {
			return bizerror.Wrap(err, bizerror.ConfigError,
				fmt.Sprintf("invalid grafana url: %s", c.Grafana))
		}
		c.GrafanaBaseURL = grafanaBaseURL
	}
	if c.Logs != nil {
		if err := c.Logs.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func DefaultObservabilityConfig() *Config {
	return &Config{}
}
