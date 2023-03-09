package services

import (
	"fmt"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/common/yaml"
	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model/adapter"
	"github.com/apache/dubbo-admin/pkg/admin/model/domain"
	"github.com/apache/dubbo-admin/pkg/admin/model/dto"
	"github.com/apache/dubbo-admin/pkg/admin/model/store"
	"github.com/apache/dubbo-admin/pkg/admin/util"
)

type OverrideServiceImpl struct{}

func (s *OverrideServiceImpl) SaveOverride(override *dto.DynamicConfigDTO) error {
	id := util.BuildServiceKey(override.Service, override.ServiceVersion, override.ServiceGroup)
	path := getPath(id)
	exitConfig, err := config.GetConfig(path)
	if err != nil {
		logger.Error(err)
		return err
	}

	existOverride := adapter.NewDynamicConfigDTO2OverrideDTOAdapter(override).OverrideDTO
	configs := []store.OverrideConfig{}
	if exitConfig != "" {
		exist, err := yaml.LoadYMLConfig(exitConfig)
		if err != nil {
			logger.Error(err)
			return err
		}
		err = yaml.UnmarshalYML(exist, existOverride)
		if err != nil {
			logger.Error(err)
			return err
		}
		if existOverride.Configs != nil && len(existOverride.Configs) > 0 {
			for _, overrideConfig := range existOverride.Configs {
				if constant.Configs.Contains(overrideConfig) {
					configs = append(configs, overrideConfig)
				}
			}
		}
	}
	configs = append(configs, override.Configs...)
	existOverride.Enabled = override.Enabled
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
	if override.Service != "" {
		result := convertDTOtoOldOverride(override)
		for _, o := range result {
			err := config.RegistryCenter.Register(o.ToURL())
			if err != nil {
				logger.Error(err)
				return err
			}
		}
	}

	return nil
}
func convertDTOtoOldOverride(overrideDTO *dto.DynamicConfigDTO) []*domain.Override {
	result := []*domain.Override{}
	configs := overrideDTO.Configs
	for _, config := range configs {
		if constant.Configs.Contains(config) {
			continue
		}
		apps := config.Applications
		addresses := config.Addresses
		for _, address := range addresses {
			if len(apps) > 0 {
				for _, app := range apps {
					override := &domain.Override{
						Service: overrideDTO.Service,
						Address: address,
						Enabled: overrideDTO.Enabled,
					}
					overrideDTOToParams(override, config)
					override.Application = app
					result = append(result, override)
				}
			} else {
				override := &domain.Override{
					Service: overrideDTO.Service,
					Address: address,
					Enabled: overrideDTO.Enabled,
				}
				overrideDTOToParams(override, config)
				result = append(result, override)
			}
		}
	}
	return result
}

func overrideDTOToParams(override *domain.Override, config store.OverrideConfig) {
	parameters := config.Parameters
	var params strings.Builder

	if parameters != nil {
		for key, value := range parameters {
			param := key + "=" + fmt.Sprintf("%v", value)
			params.WriteString(param)
			params.WriteString("&")
		}
	}
	s := ""
	if params.Len() > 0 {
		if params.String()[params.Len()-1] == '&' {

			s = params.String()[:(params.Len() - 1)]
		}
	}
	override.Params = s
}

func getPath(key string) string {
	key = strings.Replace(key, "/", "*", -1)
	return key + constant.ConfiguratorsCategory
}

func (s *OverrideServiceImpl) UpdateOverride(update *dto.DynamicConfigDTO) error {
	id := util.BuildServiceKey(update.Service, update.ServiceGroup, update.ServiceVersion)
	path := getPath(id)
	existConfig, err := config.GetConfig(path)
	if err != nil {
		logger.Error(err)
		return err
	}

	var overrideDto store.OverrideDTO
	exist, err := yaml.LoadYMLConfig(existConfig)
	if err != nil {
		logger.Error(err)
		return err
	}

	err = yaml.UnmarshalYML(exist, overrideDto)
	if err != nil {
		logger.Error(err)
		return err
	}
	configs := make([]store.OverrideConfig, 0)
	if overrideDto.Configs != nil {
		overrideConfigs := overrideDto.Configs
		for _, config := range overrideConfigs {
			if constant.Configs.Contains(config) {
				configs = append(configs, config)
			}
		}
	}
	configs = append(configs, update.Configs...)
	overrideDto.Configs = configs
	overrideDto.Enabled = update.Enabled
	if b, err := yaml.MarshalYML(overrideDto); err != nil {
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
	old := util.CreateFromOverride(overrideDto)
	if update.Service != "" {
		oldOverrides := convertDTOtoOldOverride(old)
		updatedOverrides := convertDTOtoOldOverride(update)
		for _, o := range oldOverrides {
			config.RegistryCenter.UnRegister(o.ToURL())
		}
		for _, o := range updatedOverrides {
			config.RegistryCenter.Register(o.ToURL())
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

	override := &store.OverrideDTO{}

	exist, err := yaml.LoadYMLConfig(conf)
	if err != nil {
		logger.Error(err)
		return err
	}

	err = yaml.UnmarshalYML(exist, override)
	if err != nil {
		logger.Error(err)
		return err
	}
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
	old := util.CreateFromOverride(*override)
	if override.Scope == constant.Service {
		overrides := convertDTOtoOldOverride(old)
		for _, o := range overrides {
			o.Enabled = true
			config.RegistryCenter.UnRegister(o.ToURL())
			o.Enabled = false
			config.RegistryCenter.Register(o.ToURL())
		}
	}

	return nil
}

func (s *OverrideServiceImpl) FindOverride(id string) (*dto.DynamicConfigDTO, error) {
	path := getPath(id)
	conf, err := config.GetConfig(path)
	if err != nil {
		return nil, err
	}

	if conf != "" {
		var overrideDto store.OverrideDTO

		exist, err := yaml.LoadYMLConfig(conf)
		if err != nil {
			logger.Error(err)
			return nil, err
		}
		yaml.UnmarshalYML(exist, overrideDto)

		dynamicConfigDto := util.CreateFromOverride(overrideDto)
		if dynamicConfigDto != nil {
			dynamicConfigDto.ID = id
			if constant.Service == overrideDto.Scope {
				// detachIdToService
				dynamicConfigDto.Service = util.GetInterface(id)
				dynamicConfigDto.ServiceGroup = util.GetGroup(id)
				dynamicConfigDto.ServiceVersion = util.GetVersion(id)
			}
		}
		return dynamicConfigDto, nil
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

	override := &store.OverrideDTO{}
	exist, err := yaml.LoadYMLConfig(conf)
	if err != nil {
		logger.Error(err)
		return err
	}
	yaml.UnmarshalYML(exist, override)

	old := util.CreateFromOverride(*override)
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
		overrides := convertDTOtoOldOverride(old)
		for _, o := range overrides {
			o.Enabled = false
			config.RegistryCenter.UnRegister(o.ToURL())
			o.Enabled = true
			config.RegistryCenter.Register(o.ToURL())
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

	var override store.OverrideDTO

	exist, err := yaml.LoadYMLConfig(conf)
	if err != nil {
		logger.Error(err)
		return err
	}
	yaml.UnmarshalYML(exist, override)

	old := util.CreateFromOverride(override)
	newConfigs := make([]store.OverrideConfig, 0)
	if override.Configs != nil && len(override.Configs) > 0 {
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
		overrides := convertDTOtoOldOverride(old)
		for _, o := range overrides {
			config.RegistryCenter.UnRegister(o.ToURL())
		}
	}

	return nil
}
