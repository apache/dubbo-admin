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
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/apache/dubbo-admin/pkg/config"
)

// MetricDashboardConfig are grafana dashboards for metrics display
type MetricDashboardConfig struct {
	config.BaseConfig
	// application level metrics panel
	Application DashboardConfig `json:"application"`
	// instance level metrics panel
	Instance DashboardConfig `json:"instance"`
	// service level metrics panel
	Service DashboardConfig `json:"service"`
}

func (c *MetricDashboardConfig) PostProcess() error {
	return multierr.Combine(
		c.Application.PostProcess(),
		c.Instance.PostProcess(),
		c.Service.PostProcess(),
	)
}

func (c *MetricDashboardConfig) Validate() error {
	return multierr.Combine(
		c.Application.Validate(),
		c.Instance.Validate(),
		c.Service.Validate(),
	)
}

// TraceDashboardConfig are grafana dashboards for traces display
type TraceDashboardConfig struct {
	config.BaseConfig
	// application level traces panel
	Application DashboardConfig `json:"application"`
	// instance level traces panel
	Instance DashboardConfig `json:"instance"`
	// service level traces panel
	Service DashboardConfig `json:"service"`
}

func (c *TraceDashboardConfig) PostProcess() error {
	return multierr.Combine(
		c.Application.PostProcess(),
		c.Instance.PostProcess(),
		c.Service.PostProcess(),
	)
}

func (c *TraceDashboardConfig) Validate() error {
	return multierr.Combine(
		c.Application.Validate(),
		c.Instance.Validate(),
		c.Service.Validate(),
	)
}

// DashboardConfig grafana dashboard config TODO add dynamic variables
type DashboardConfig struct {
	config.BaseConfig
	BaseURL string `json:"baseURL"`
}

func (c *DashboardConfig) PostProcess() error {
	// trim suffix "/" of dashboard url
	c.BaseURL = strings.TrimSuffix(c.BaseURL, "/")
	return nil
}

func (c *DashboardConfig) Validate() error {
	_, err := url.Parse(c.BaseURL)
	if err != nil {
		return errors.Wrap(err, "invalid url"+c.BaseURL)
	}
	return nil
}

// PrometheusConfig is used to query metrics data for frontend
type PrometheusConfig string
