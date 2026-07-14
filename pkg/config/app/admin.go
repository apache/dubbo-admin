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
	"github.com/duke-git/lancet/v2/slice"
	"go.uber.org/multierr"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/config"
	"github.com/apache/dubbo-admin/pkg/config/console"
	"github.com/apache/dubbo-admin/pkg/config/diagnostics"
	"github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/config/engine"
	"github.com/apache/dubbo-admin/pkg/config/eventbus"
	"github.com/apache/dubbo-admin/pkg/config/log"
	"github.com/apache/dubbo-admin/pkg/config/observability"
	"github.com/apache/dubbo-admin/pkg/config/store"
	"github.com/apache/dubbo-admin/pkg/config/versioning"
)

type AdminConfig struct {
	config.BaseConfig
	// Log configuration
	Log *log.Config `json:"log" yaml:"log"`
	// Diagnostics configuration
	Diagnostics *diagnostics.Config `json:"diagnostics,omitempty" yaml:"diagnostics"`
	// Observability configuration
	Observability *observability.Config `json:"observability" yaml:"observability"`
	// Console configuration
	Console *console.Config `json:"console" yaml:"console"`
	// Store configuration
	Store *store.Config `json:"store" yaml:"store"`
	// Discovery configuration
	Discovery []*discovery.Config `json:"discovery" yaml:"discovery"`
	// Engine configuration
	Engine *engine.Config `json:"engine" yaml:"engine"`
	// EventBus configuration
	EventBus *eventbus.Config `json:"eventBus,omitempty" yaml:"eventBus,omitempty"`
	// RuleVersioning records lightweight audit history for governor-managed traffic rules.
	// Live rule state remains in ResourceManager/registry.
	RuleVersioning *versioning.Config `json:"ruleVersioning,omitempty" yaml:"ruleVersioning,omitempty"`
}

var _ = &AdminConfig{}

var DefaultAdminConfig = func() AdminConfig {
	eventBusCfg := eventbus.Default()
	return AdminConfig{
		Log:            log.DefaultLogConfig(),
		Store:          store.DefaultStoreConfig(),
		Engine:         engine.DefaultResourceEngineConfig(),
		Observability:  observability.DefaultObservabilityConfig(),
		Diagnostics:    diagnostics.DefaultDiagnosticsConfig(),
		Console:        console.DefaultConsoleConfig(),
		EventBus:       &eventBusCfg,
		RuleVersioning: versioning.Default(),
	}
}

func (c *AdminConfig) Sanitize() {
	c.Engine.Sanitize()
	for _, d := range c.Discovery {
		d.Sanitize()
	}
	c.Store.Sanitize()
	c.Console.Sanitize()
	c.Observability.Sanitize()
	c.Diagnostics.Sanitize()
	c.Log.Sanitize()
	if c.RuleVersioning == nil {
		c.RuleVersioning = versioning.Default()
	}
	c.RuleVersioning.Sanitize()
}

func (c *AdminConfig) PreProcess() error {
	discoveryPreProcess := func() error {
		for _, d := range c.Discovery {
			if err := d.PreProcess(); err != nil {
				return err
			}
		}
		return nil
	}
	if c.RuleVersioning == nil {
		c.RuleVersioning = versioning.Default()
	}
	return multierr.Combine(
		c.Engine.PreProcess(),
		discoveryPreProcess(),
		c.Store.PreProcess(),
		c.Console.PreProcess(),
		c.Observability.PreProcess(),
		c.Diagnostics.PreProcess(),
		c.Log.PreProcess(),
		c.RuleVersioning.PreProcess(),
	)
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
	if c.RuleVersioning == nil {
		c.RuleVersioning = versioning.Default()
	}
	return multierr.Combine(
		c.Engine.PostProcess(),
		discoveryPostProcess(),
		c.Store.PostProcess(),
		c.Console.PostProcess(),
		c.Observability.PostProcess(),
		c.Diagnostics.PostProcess(),
		c.Log.PostProcess(),
		c.RuleVersioning.PostProcess(),
	)
}

func (c *AdminConfig) Validate() error {
	if c.Log == nil {
		c.Log = log.DefaultLogConfig()
	} else if err := c.Log.Validate(); err != nil {
		return bizerror.Wrap(err, bizerror.ConfigError, "log config validation failed")
	}
	if c.Store == nil {
		c.Store = store.DefaultStoreConfig()
	} else if err := c.Store.Validate(); err != nil {
		return bizerror.Wrap(err, bizerror.ConfigError, "store config validation failed")
	}
	if c.Diagnostics == nil {
		c.Diagnostics = diagnostics.DefaultDiagnosticsConfig()
	} else if err := c.Diagnostics.Validate(); err != nil {
		return bizerror.Wrap(err, bizerror.ConfigError, "diagnostics config validation failed")
	}
	if c.Console == nil {
		c.Console = console.DefaultConsoleConfig()
	} else if err := c.Console.Validate(); err != nil {
		return bizerror.Wrap(err, bizerror.ConfigError, "console config validation failed")
	}
	if c.Observability == nil {
		c.Observability = observability.DefaultObservabilityConfig()
	} else if err := c.Observability.Validate(); err != nil {
		return bizerror.Wrap(err, bizerror.ConfigError, "observability config validation failed")
	}
	if c.Discovery == nil || len(c.Discovery) == 0 {
		return bizerror.New(bizerror.ConfigError, "discover config is needed")
	}
	for _, d := range c.Discovery {
		if err := d.Validate(); err != nil {
			return bizerror.Wrap(err, bizerror.ConfigError, "discovery config validation failed")
		}
	}
	discoveryIDList := slice.Map(c.Discovery, func(index int, item *discovery.Config) string {
		return item.ID
	})
	if len(discoveryIDList) != len(slice.Unique(discoveryIDList)) {
		return bizerror.New(bizerror.ConfigError, "discovery id must be unique")
	}
	if c.Engine == nil {
		c.Engine = engine.DefaultResourceEngineConfig()
	} else if err := c.Engine.Validate(); err != nil {
		return bizerror.Wrap(err, bizerror.ConfigError, "engine config validation failed")
	}
	if c.EventBus == nil {
		cfg := eventbus.Default()
		c.EventBus = &cfg
	} else if err := c.EventBus.Validate(); err != nil {
		return bizerror.Wrap(err, bizerror.ConfigError, "event bus config validation failed")
	}
	if c.RuleVersioning == nil {
		c.RuleVersioning = versioning.Default()
	} else if err := c.RuleVersioning.Validate(); err != nil {
		return bizerror.Wrap(err, bizerror.ConfigError, "versioning config validation failed")
	}
	return nil
}

// FindDiscovery finds the DiscoveryConfig by id, returns nil if not found
func (c *AdminConfig) FindDiscovery(id string) *discovery.Config {
	for _, d := range c.Discovery {
		if d.ID == id {
			return d
		}
	}
	return nil
}

// Meshes return the mesh id list of discoveries
func (c *AdminConfig) Meshes() []string {
	return slice.Map(c.Discovery, func(index int, item *discovery.Config) string {
		return item.ID
	})
}
