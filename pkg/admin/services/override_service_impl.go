package services

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/yaml"
	"fmt"
	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/model/adapter"
	"github.com/apache/dubbo-admin/pkg/admin/model/domain"
	"github.com/apache/dubbo-admin/pkg/admin/model/dto"
	"github.com/apache/dubbo-admin/pkg/admin/model/store"
	"github.com/apache/dubbo-admin/pkg/admin/util"
	_ "github.com/apache/dubbo-admin/pkg/admin/util"
	set "github.com/dubbogo/gost/container/set"
	"log"
	"reflect"
	"strings"
)

type OverrideServiceImpl struct {
}

var HashSet set.HashSet

// 还有registry
//var regis registry.MockRegistry

func (s *OverrideServiceImpl) SaveOverride(override dto.DynamicConfigDTO) {

	HashSet = set.HashSet{
		Items: make(map[interface{}]struct{}),
	}
	convert := util.Convert{}
	id := convert.GetIdFromDTO(override.BaseDTO)

	path := getPath(id.(string))
	fmt.Println("saveOve哒哒哒啊")
	exitConfig, err := config.GetConfig(path)
	if err != nil {
		log.Fatal("", err)
	}
	configs := []store.OverrideConfig{}

	adapt, _ := adapter.NewDynamicConfigDTO2OverrideDTOAdapter(override)
	existOverride := adapt.OverrideDTO
	//parser := properties.Parser()

	if reflect.ValueOf(exitConfig).IsNil() {
		exist, err := yaml.LoadYMLConfig(exitConfig)
		if err != nil {
			log.Fatal("", err)
		}
		err = yaml.UnmarshalYML(exist, existOverride)
		if err != nil {
			log.Fatal("", err)
		}
		if existOverride.GetConfigs() == nil {
			for _, overrideConfig := range existOverride.Configs {
				if HashSet.Contains(overrideConfig) {
					configs = append(configs, overrideConfig)
				}
			}
		}
	}
	configs = append(configs, override.Configs...)
	existOverride.Enabled = override.Enabled
	existOverride.Configs = configs
	if b, err := yaml.MarshalYML(existOverride); err == nil {
		err := config.SetConfig(path, string(b))
		if err != nil {
			return
		}
	} else {
		panic(err)
	}
	//放在configCenter初始化的地方

	if override.Service != "" {
		result := convertDTOtoOldOverride(&override)
		for _, o := range result {
			//o.ToURL().AddParam(constant.CompatibleConfigKey, strconv.FormatBool(true))
			err := config.RegistryCenter.Register(o.ToURL())
			if err != nil {
				return
			}
		}
	}
}
func convertDTOtoOldOverride(overrideDTO *dto.DynamicConfigDTO) []*domain.Override {
	result := []*domain.Override{}
	configs := overrideDTO.Configs
	for _, config := range configs {
		if HashSet.Contains(config) {
			continue
		}
		apps := config.Applications
		addresses := config.Addresses
		for _, address := range addresses {
			if apps != nil && len(apps) > 0 {
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
	//ConfiguratorRuleSuffix
	return key + constant.ConfiguratorSuffix

}

//func (s *OverrideServiceImpl) FindOverride(id string) dto.DynamicConfigDTO {
//	return dto.DynamicConfigDTO{}
//}

func (s *OverrideServiceImpl) UpdateOverride(update dto.DynamicConfigDTO) {
	convert := util.Convert{}
	id := convert.GetIdFromDTO(update)
	path := getPath(id.(string))
	exitConfig, err := config.GetConfig(path)
	if exitConfig == "" {
		log.Fatal("", err)
		// throw exception
	}

	var overrideDto dto.OverrideDTO
	exist, err := yaml.LoadYMLConfig(exitConfig)
	if err != nil {
		log.Fatal("", err)
	}

	err = yaml.UnmarshalYML(exist, overrideDto)
	if err != nil {
		log.Fatal("", err)
	}
	old := util.CreateFromOverride(overrideDto)
	configs := make([]store.OverrideConfig, 0)
	if overrideDto.Configs != nil {
		overrideConfigs := overrideDto.Configs
		for _, config := range overrideConfigs {
			if HashSet.Contains(config) {
				configs = append(configs, config)
			}
		}
	}
	configs = append(configs, update.Configs...)
	overrideDto.Configs = configs
	overrideDto.Enabled = update.Enabled
	if b, err := yaml.MarshalYML(overrideDto); err == nil {
		err := config.SetConfig(path, string(b))
		if err != nil {
			return
		}
	} else {
		panic(err)
	}

	// for 2.6
	if update.Service != "" {
		oldOverrides := convertDTOtoOldOverride(old)
		updatedOverrides := convertDTOtoOldOverride(&update)
		for _, o := range oldOverrides {
			config.RegistryCenter.UnRegister(o.ToURL())
		}
		for _, o := range updatedOverrides {
			config.RegistryCenter.Register(o.ToURL())
		}
	}

}

func (s *OverrideServiceImpl) DisableOverride(id string) {
	if id == "" {
		// throw exception
	}
	path := getPath(id)
	if conf, _ := config.GetConfig(path); conf == "" {
		// throw exception
	}

	conf, err := config.GetConfig(path)
	if err != nil {
		log.Fatal(err)
	}
	override := &dto.OverrideDTO{}

	exist, err := yaml.LoadYMLConfig(conf)
	if err != nil {
		log.Fatal("", err)
	}

	err = yaml.UnmarshalYML(exist, override)
	if err != nil {
		log.Fatal("", err)
	}

	old := util.CreateFromOverride(*override)
	override.Enabled = false
	if b, err := yaml.MarshalYML(override); err == nil {

		err := config.SetConfig(path, string(b))
		if err != nil {
			return
		}
	} else {
		panic(err)
	}
	// for 2.6
	if override.Scope == constant.ServiceKey {
		overrides := convertDTOtoOldOverride(old)
		for _, o := range overrides {
			o.Enabled = true
			config.RegistryCenter.UnRegister(o.ToURL())
			o.Enabled = false
			config.RegistryCenter.Register(o.ToURL())
		}
	}
}

func (s *OverrideServiceImpl) FindOverride(id string) {
	if id == "" {
		// throw exception
	}
	path := getPath(id)
	conf, _ := config.GetConfig(path)
	var overrideDto dto.OverrideDTO
	if conf != "" {
		exist, err := yaml.LoadYMLConfig(conf)
		if err != nil {
			log.Fatal("", err)
		}
		yaml.UnmarshalYML(exist, overrideDto)

		old := util.CreateFromOverride(overrideDto)
		overrideDto.Enabled = false
		if b, err := yaml.MarshalYML(overrideDto); err == nil {
			err := config.SetConfig(path, string(b))
			if err != nil {
				return
			}
		} else {
			panic(err)
		}
		if overrideDto.Scope == constant.ServiceKey {
			overrides := convertDTOtoOldOverride(old)
			for _, o := range overrides {
				o.Enabled = true
				config.RegistryCenter.UnRegister(o.ToURL())
				o.Enabled = false
				config.RegistryCenter.Register(o.ToURL())
			}
		}
	}
}

//func (s *OverrideServiceImpl) EnableOverride(id string) {
//	if id == "" {
//		// throw exception
//	}
//
//	path := getPath(id)
//	config, _ := config.GetConfig(path)
//	if config == "" {
//		// throw exception
//	}
//
//	override := &dto.OverrideDTO{}
//	YamlParser.LoadObject(config, override)
//	old := OverrideUtils.CreateFromOverride(*override)
//	override.Enabled = true
//	dynamicConfiguration.SetConfig(path, YamlParser.DumpObject(override))
//
//	//2.6
//	if override.Scope == Constants.SERVICE {
//		overrides := convertDTOtoOldOverride(old)
//		for _, o := range overrides {
//			o.Enabled = false
//			registry.Unregister(o.ToUrl().AddParameter(Constants.COMPATIBLE_CONFIG, true))
//			o.Enabled = true
//			registry.Register(o.ToUrl().AddParameter(Constants.COMPATIBLE_CONFIG, true))
//		}
//	}
//}

//func (s *OverrideServiceImpl) DeleteOverride(id string) {
//	if id == "" {
//		// throw exception
//	}
//	path := getPath(id)
//	config := dynamicConfiguration.getConfig(path)
//	if config == "" {
//		// throw exception
//	}
//	overrideDTO := YamlParser.loadObject(config, OverrideDTO{})
//	old := OverrideUtils.createFromOverride(overrideDTO)
//	newConfigs := make([]OverrideConfig, 0)
//	if overrideDTO.Configs != nil && len(overrideDTO.Configs) > 0 {
//		for _, overrideConfig := range overrideDTO.Configs {
//			if Constants.CONFIGS.Contains(overrideConfig.Type) {
//				newConfigs = append(newConfigs, overrideConfig)
//			}
//		}
//		if len(newConfigs) == 0 {
//			dynamicConfiguration.deleteConfig(path)
//		} else {
//			overrideDTO.Configs = newConfigs
//			dynamicConfiguration.setConfig(path, YamlParser.dumpObject(overrideDTO))
//		}
//	} else {
//		dynamicConfiguration.deleteConfig(path)
//	}
//
//	// for 2.6
//	if overrideDTO.Scope == Constants.SERVICE {
//		overrides := convertDTOtoOldOverride(old)
//		for _, o := range overrides {
//			registry.unregister(o.toUrl().addParameter(Constants.COMPATIBLE_CONFIG, true))
//		}
//	}
//
//}
