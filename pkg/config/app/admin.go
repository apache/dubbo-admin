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

package app

import (
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/apache/dubbo-admin/pkg/config"
	"github.com/apache/dubbo-admin/pkg/config/console"
	"github.com/apache/dubbo-admin/pkg/config/diagnostics"
	"github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/config/engine"
	"github.com/apache/dubbo-admin/pkg/config/mode"
	"github.com/apache/dubbo-admin/pkg/config/store"
)

type AdminConfig struct {
	config.BaseConfig
	// Mode in which dubbo admin is running. Available values are: "test", "global", "zone"
	Mode mode.Mode `json:"mode" envconfig:"DUBBO_MODE"`
	// Diagnostics configuration
	Diagnostics *diagnostics.Config `json:"diagnostics,omitempty"`
	// Console configuration
	Console *console.Config `json:"admin"`
	// Store configuration
	Store *store.Config `json:"store"`
	// Discovery configuration
	Discovery []*discovery.Config `json:"discovery"`
	// Engine configuration
	Engine *engine.Config `json:"engine"`
}

var _ = &AdminConfig{}

func (c *AdminConfig) Sanitize() {
	c.Engine.Sanitize()
	for _, d := range c.Discovery {
		d.Sanitize()
	}
	c.Store.Sanitize()
	c.Console.Sanitize()
	c.Diagnostics.Sanitize()
}

func (c *AdminConfig) PostProcess() error {
	discoveryPostProcess := func() error {
		for _, d := range c.Discovery {
			if err := d.PostProcess(); err != nil {
				return err
			}
		}
		return nil
	}
	return multierr.Combine(
		c.Engine.PostProcess(),
		discoveryPostProcess(),
		c.Store.PostProcess(),
		c.Console.PostProcess(),
		c.Diagnostics.PostProcess(),
	)
}

var DefaultAdminConfig = func() AdminConfig {
	return AdminConfig{
		Mode:        mode.Zone,
		Store:       store.DefaultStoreConfig(),
		Engine:      engine.DefaultResourceEngineConfig(),
		Diagnostics: diagnostics.DefaultDiagnosticsConfig(),
		Console:     console.DefaultConsoleConfig(),
	}
}

func (c *AdminConfig) Validate() error {
	if err := mode.ValidateMode(c.Mode); err != nil {
		return errors.Wrap(err, "Mode Config validation failed")
	}
	if c.Store == nil {
		c.Store = store.DefaultStoreConfig()
	} else if err := c.Store.Validate(); err != nil {
		return errors.Wrap(err, "Store Config validation failed")
	}
	if c.Diagnostics == nil {
		c.Diagnostics = diagnostics.DefaultDiagnosticsConfig()
	} else if err := c.Diagnostics.Validate(); err != nil {
		return errors.Wrap(err, "Diagnostics Config validation failed")
	}
	if c.Console == nil {
		c.Console = console.DefaultConsoleConfig()
	} else if err := c.Console.Validate(); err != nil {
		return errors.Wrap(err, "Admin validation failed")
	}
	return nil
}
