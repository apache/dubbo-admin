// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package services

import (
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/common/yaml"
	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/util"
)

type OverrideServiceImpl struct{}

func (s *OverrideServiceImpl) SaveOverride(dynamicConfig *model.DynamicConfig) error {
	id := util.BuildServiceKey(dynamicConfig.Service, dynamicConfig.ServiceVersion, dynamicConfig.ServiceGroup)
	path := getPath(id)
	existConfig, err := config.GetConfig(path)
	if err != nil {
		logger.Error(err)
		return err
	}

	existOverride := dynamicConfig.ToOverride()
	configs := []model.OverrideConfig{}
	if existConfig != "" {
		b, err := yaml.LoadYMLConfig(existConfig)
		if err != nil {
			logger.Error(err)
			return err
		}
		err = yaml.UnmarshalYML(b, existOverride)
		if err != nil {
			logger.Error(err)
			return err
		}
		if len(existOverride.Configs) > 0 {
			for _, c := range existOverride.Configs {
				if constant.Configs.Contains(c) {
					configs = append(configs, c)
				}
			}
		}
	}
	configs = append(configs, dynamicConfig.Configs...)
	existOverride.Enabled = dynamicConfig.Enabled
	existOverride.Configs = configs
	if b, err := yaml.MarshalYML(existOverride); err != nil {
		logger.Error(err)
		return err
	} else {
		err := config.SetConfig(path, string(b))
		if err != nil {
			logger.Error(err)
			return err
		}
	}

	// for 2.6
	if dynamicConfig.Service != "" {
		result := dynamicConfig.ToOldOverride()
		for _, o := range result {
			url, err := util.OldOverride2URL(o)
			if err != nil {
				logger.Error(err)
				return err
			}
			err = config.RegistryCenter.Register(url)
			if err != nil {
				logger.Error(err)
				return err
			}
		}
	}

	return nil
}

// TODO: check key type
func getPath(key string) string {
	key = strings.Replace(key, "/", "*", -1)
	return key + constant.ConfiguratorsCategory
}

func (s *OverrideServiceImpl) UpdateOverride(update *model.DynamicConfig) error {
	id := util.BuildServiceKey(update.Service, update.ServiceGroup, update.ServiceVersion)
	path := getPath(id)
	existConfig, err := config.GetConfig(path)
	if err != nil {
		logger.Error(err)
		return err
	}

	var override model.Override
	b, err := yaml.LoadYMLConfig(existConfig)
	if err != nil {
		logger.Error(err)
		return err
	}
	err = yaml.UnmarshalYML(b, override)
	if err != nil {
		logger.Error(err)
		return err
	}
	old := override.ToDynamicConfig()

	configs := make([]model.OverrideConfig, 0)
	if override.Configs != nil {
		overrideConfigs := override.Configs
		for _, config := range overrideConfigs {
			if constant.Configs.Contains(config) {
				configs = append(configs, config)
			}
		}
	}
	configs = append(configs, update.Configs...)
	override.Configs = configs
	override.Enabled = update.Enabled
	if b, err := yaml.MarshalYML(override); err != nil {
		logger.Error(err)
		return err
	} else {
		err := config.SetConfig(path, string(b))
		if err != nil {
			logger.Error(err)
			return err
		}
	}

	// for 2.6
	if update.Service != "" {
		oldOverrides := old.ToOldOverride()
		updatedOverrides := update.ToOldOverride()
		for _, o := range oldOverrides {
			url, err := util.OldOverride2URL(o)
			if err != nil {
				return err
			}
			config.RegistryCenter.UnRegister(url)
		}
		for _, o := range updatedOverrides {
			url, err := util.OldOverride2URL(o)
			if err != nil {
				return err
			}
			config.RegistryCenter.Register(url)
		}
	}

	return nil
}

func (s *OverrideServiceImpl) DisableOverride(id string) error {
	path := getPath(id)

	conf, err := config.GetConfig(path)
	if err != nil {
		logger.Error(err)
		return err
	}

	override := &model.Override{}

	b, err := yaml.LoadYMLConfig(conf)
	if err != nil {
		logger.Error(err)
		return err
	}
	err = yaml.UnmarshalYML(b, override)
	if err != nil {
		logger.Error(err)
		return err
	}
	old := override.ToDynamicConfig()
	override.Enabled = false

	if b, err := yaml.MarshalYML(override); err != nil {
		logger.Error(err)
		return err
	} else {
		err := config.SetConfig(path, string(b))
		if err != nil {
			return err
		}
	}

	// for 2.6
	if override.Scope == constant.Service {
		overrides := old.ToOldOverride()
		for _, o := range overrides {
			o.Enabled = true
			url, err := util.OldOverride2URL(o)
			if err != nil {
				logger.Error(err)
				return err
			}
			config.RegistryCenter.UnRegister(url)

			o.Enabled = false
			url, err = util.OldOverride2URL(o)
			if err != nil {
				logger.Error(err)
				return err
			}
			config.RegistryCenter.Register(url)
		}
	}

	return nil
}

func (s *OverrideServiceImpl) FindOverride(id string) (*model.DynamicConfig, error) {
	path := getPath(id)
	conf, err := config.GetConfig(path)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	if conf != "" {
		var override model.Override

		b, err := yaml.LoadYMLConfig(conf)
		if err != nil {
			logger.Error(err)
			return nil, err
		}
		yaml.UnmarshalYML(b, override)

		dynamicConfig := override.ToDynamicConfig()
		if dynamicConfig != nil {
			dynamicConfig.ID = id
			if constant.Service == override.Scope {
				dynamicConfig.Service = util.GetInterface(id)
				dynamicConfig.ServiceGroup = util.GetGroup(id)
				dynamicConfig.ServiceVersion = util.GetVersion(id)
			}
		}
		return dynamicConfig, nil
	}

	return nil, nil
}

func (s *OverrideServiceImpl) EnableOverride(id string) error {
	path := getPath(id)
	conf, err := config.GetConfig(path)
	if err != nil {
		logger.Error(err)
		return err
	}

	override := &model.Override{}
	b, err := yaml.LoadYMLConfig(conf)
	if err != nil {
		logger.Error(err)
		return err
	}
	err = yaml.UnmarshalYML(b, override)
	if err != nil {
		logger.Error(err)
		return err
	}

	old := override.ToDynamicConfig()
	override.Enabled = true
	if b, err := yaml.MarshalYML(override); err != nil {
		logger.Error(err)
		return err
	} else {
		err := config.SetConfig(path, string(b))
		if err != nil {
			logger.Error(err)
			return err
		}
	}

	// for 2.6
	if override.Scope == constant.Service {
		overrides := old.ToOldOverride()
		for _, o := range overrides {
			o.Enabled = false
			url, err := util.OldOverride2URL(o)
			if err != nil {
				return err
			}
			config.RegistryCenter.UnRegister(url)

			o.Enabled = true
			url, err = util.OldOverride2URL(o)
			if err != nil {
				return err
			}
			config.RegistryCenter.Register(url)
		}
	}

	return nil
}

func (s *OverrideServiceImpl) DeleteOverride(id string) error {
	path := getPath(id)
	conf, err := config.GetConfig(path)
	if err != nil {
		logger.Error(err)
		return err
	}

	var override model.Override
	b, err := yaml.LoadYMLConfig(conf)
	if err != nil {
		logger.Error(err)
		return err
	}
	err = yaml.UnmarshalYML(b, override)
	if err != nil {
		logger.Error(err)
		return err
	}

	old := override.ToDynamicConfig()
	newConfigs := make([]model.OverrideConfig, 0)
	if len(override.Configs) > 0 {
		for _, overrideConfig := range override.Configs {
			if constant.Configs.Contains(overrideConfig.Type) {
				newConfigs = append(newConfigs, overrideConfig)
			}
		}
		if len(newConfigs) == 0 {
			err := config.DeleteConfig(path)
			if err != nil {
				logger.Error(err)
				return err
			}
		} else {
			override.Configs = newConfigs
			if b, err := yaml.MarshalYML(override); err != nil {
				logger.Error(err)
				return err
			} else {
				err := config.SetConfig(path, string(b))
				if err != nil {
					logger.Error(err)
					return err
				}
			}
		}
	} else {
		err := config.DeleteConfig(path)
		if err != nil {
			logger.Error(err)
			return err
		}
	}

	// for 2.6
	if override.Scope == constant.Service {
		overrides := old.ToOldOverride()
		for _, o := range overrides {
			url, err := util.OldOverride2URL(o)
			if err != nil {
				logger.Error(err)
				return err
			}
			config.RegistryCenter.UnRegister(url)
		}
	}

	return nil
}
