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

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/dubbogo/gost/encoding/yaml"

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/model/util"
	util2 "github.com/apache/dubbo-admin/pkg/admin/util"
)

type OverrideServiceImpl struct{}

func (s *OverrideServiceImpl) SaveOverride(dynamicConfig *model.DynamicConfig) error {
	id := util2.BuildServiceKey(dynamicConfig.Base.Application, dynamicConfig.Base.Service, dynamicConfig.Base.ServiceVersion, dynamicConfig.Base.ServiceGroup)
	path := GetOverridePath(id)
	existConfig, err := config.Governance.GetConfig(path)
	if err != nil {
		if _, ok := err.(*config.RuleNotFound); !ok {
			logger.Logger().Error(err.Error())
			return err
		}
	}

	existOverride := dynamicConfig.ToOverride()
	configs := []model.OverrideConfig{}
	if existConfig != "" {
		err := yaml.UnmarshalYML([]byte(existConfig), existOverride)
		if err != nil {
			logger.Logger().Error(err.Error())
			return err
		}
		if len(existOverride.Configs) > 0 {
			for _, c := range existOverride.Configs {
				if constant.Configs.Contains(c.Type) {
					configs = append(configs, c)
				}
			}
		}
	}
	configs = append(configs, dynamicConfig.Configs...)
	existOverride.Enabled = dynamicConfig.Enabled
	existOverride.Configs = configs
	if b, err := yaml.MarshalYML(existOverride); err != nil {
		logger.Logger().Error(err.Error())
		return err
	} else {
		err := config.Governance.SetConfig(path, string(b))
		if err != nil {
			logger.Logger().Error(err.Error())
			return err
		}
	}

	// for 2.6
	if dynamicConfig.Service != "" {
		result := dynamicConfig.ToOldOverride()
		for _, o := range result {
			url, err := util.OldOverride2URL(o)
			if err != nil {
				logger.Logger().Error(err.Error())
				return err
			}
			err = config.Governance.Register(url)
			if err != nil {
				logger.Logger().Error(err.Error())
				return err
			}
		}
	}

	return nil
}

func (s *OverrideServiceImpl) UpdateOverride(update *model.DynamicConfig) error {
	id := util2.BuildServiceKey(update.Base.Application, update.Base.Service, update.Base.ServiceVersion, update.Base.ServiceGroup)
	path := GetOverridePath(id)
	existConfig, err := config.Governance.GetConfig(path)
	if err != nil {
		logger.Logger().Error(err.Error())
		return err
	}

	override := &model.Override{}
	err = yaml.UnmarshalYML([]byte(existConfig), override)
	if err != nil {
		logger.Logger().Error(err.Error())
		return err
	}
	old := override.ToDynamicConfig()

	configs := make([]model.OverrideConfig, 0)
	if len(override.Configs) > 0 {
		for _, c := range override.Configs {
			if constant.Configs.Contains(c.Type) {
				configs = append(configs, c)
			}
		}
	}
	configs = append(configs, update.Configs...)
	override.Configs = configs
	override.Enabled = update.Enabled
	if b, err := yaml.MarshalYML(override); err != nil {
		logger.Logger().Error(err.Error())
		return err
	} else {
		err := config.Governance.SetConfig(path, string(b))
		if err != nil {
			logger.Logger().Error(err.Error())
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
			config.Governance.UnRegister(url)
		}
		for _, o := range updatedOverrides {
			url, err := util.OldOverride2URL(o)
			if err != nil {
				return err
			}
			config.Governance.Register(url)
		}
	}

	return nil
}

func (s *OverrideServiceImpl) DisableOverride(id string) error {
	path := GetOverridePath(id)

	conf, err := config.Governance.GetConfig(path)
	if err != nil {
		logger.Logger().Error(err.Error())
		return err
	}

	override := &model.Override{}
	err = yaml.UnmarshalYML([]byte(conf), override)
	if err != nil {
		logger.Logger().Error(err.Error())
		return err
	}
	old := override.ToDynamicConfig()
	override.Enabled = false

	if b, err := yaml.MarshalYML(override); err != nil {
		logger.Logger().Error(err.Error())
		return err
	} else {
		err := config.Governance.SetConfig(path, string(b))
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
				logger.Logger().Error(err.Error())
				return err
			}
			config.Governance.UnRegister(url)

			o.Enabled = false
			url, err = util.OldOverride2URL(o)
			if err != nil {
				logger.Logger().Error(err.Error())
				return err
			}
			config.Governance.Register(url)
		}
	}

	return nil
}

func (s *OverrideServiceImpl) FindOverride(id string) (*model.DynamicConfig, error) {
	path := GetOverridePath(id)
	conf, err := config.Governance.GetConfig(path)
	if err != nil {
		logger.Logger().Error(err.Error())
		return nil, err
	}

	if conf != "" {
		override := &model.Override{}
		err := yaml.UnmarshalYML([]byte(conf), override)
		if err != nil {
			logger.Logger().Error(err.Error())
			return nil, err
		}

		dynamicConfig := override.ToDynamicConfig()
		if dynamicConfig != nil {
			dynamicConfig.ID = id
			if constant.Service == override.Scope {
				dynamicConfig.Service = util2.GetInterface(id)
				dynamicConfig.ServiceGroup = util2.GetGroup(id)
				dynamicConfig.ServiceVersion = util2.GetVersion(id)
			}
		}
		return dynamicConfig, nil
	}

	return nil, nil
}

func (s *OverrideServiceImpl) EnableOverride(id string) error {
	path := GetOverridePath(id)
	conf, err := config.Governance.GetConfig(path)
	if err != nil {
		logger.Logger().Error(err.Error())
		return err
	}

	override := &model.Override{}
	err = yaml.UnmarshalYML([]byte(conf), override)
	if err != nil {
		logger.Logger().Error(err.Error())
		return err
	}

	old := override.ToDynamicConfig()
	override.Enabled = true
	if b, err := yaml.MarshalYML(override); err != nil {
		logger.Logger().Error(err.Error())
		return err
	} else {
		err := config.Governance.SetConfig(path, string(b))
		if err != nil {
			logger.Logger().Error(err.Error())
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
			config.Governance.UnRegister(url)

			o.Enabled = true
			url, err = util.OldOverride2URL(o)
			if err != nil {
				return err
			}
			config.Governance.Register(url)
		}
	}

	return nil
}

func (s *OverrideServiceImpl) DeleteOverride(id string) error {
	path := GetOverridePath(id)
	conf, err := config.Governance.GetConfig(path)
	if err != nil {
		logger.Logger().Error(err.Error())
		return err
	}

	override := &model.Override{}
	err = yaml.UnmarshalYML([]byte(conf), override)
	if err != nil {
		logger.Logger().Error(err.Error())
		return err
	}
	old := override.ToDynamicConfig()

	if len(override.Configs) > 0 {
		newConfigs := make([]model.OverrideConfig, 0)
		for _, c := range override.Configs {
			if constant.Configs.Contains(c.Type) {
				newConfigs = append(newConfigs, c)
			}
		}
		if len(newConfigs) == 0 {
			err := config.Governance.DeleteConfig(path)
			if err != nil {
				logger.Logger().Error(err.Error())
				return err
			}
		} else {
			override.Configs = newConfigs
			if b, err := yaml.MarshalYML(override); err != nil {
				logger.Logger().Error(err.Error())
				return err
			} else {
				err := config.Governance.SetConfig(path, string(b))
				if err != nil {
					logger.Logger().Error(err.Error())
					return err
				}
			}
		}
	} else {
		err := config.Governance.DeleteConfig(path)
		if err != nil {
			logger.Logger().Error(err.Error())
			return err
		}
	}

	// for 2.6
	if override.Scope == constant.Service {
		overrides := old.ToOldOverride()
		for _, o := range overrides {
			url, err := util.OldOverride2URL(o)
			if err != nil {
				logger.Logger().Error(err.Error())
				return err
			}
			config.Governance.UnRegister(url)
		}
	}

	return nil
}

func GetOverridePath(key string) string {
	key = strings.Replace(key, "/", "*", -1)
	return key + constant.ConfiguratorRuleSuffix
}
