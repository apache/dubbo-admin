package impl

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/yaml"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"fmt"
	"github.com/apache/dubbo-admin/pkg/admin/model/adapter"
	"github.com/apache/dubbo-admin/pkg/admin/model/domain"
	"log"

	//"github.com/apache/dubbo-admin/cmd"
	//"admin/model/domain"
	"github.com/apache/dubbo-admin/pkg/admin/model/dto"
	"github.com/apache/dubbo-admin/pkg/admin/model/store"
	"github.com/apache/dubbo-admin/pkg/admin/util"
	_ "github.com/apache/dubbo-admin/pkg/admin/util"
	set "github.com/dubbogo/gost/container/set"
	//"log"
	"reflect"
	"strings"
)

type OverrideServiceImpl struct {
}

// SaveOverride /**   应该是DynamicConfiguration
var dynamicConfiguration config_center.MockDynamicConfiguration
var HashSet set.HashSet

func (s *OverrideServiceImpl) SaveOverride(override dto.DynamicConfigDTO) {
	HashSet = set.HashSet{
		Items: make(map[interface{}]struct{}),
	}
	// 怎么设置成动态的  DynamicConfiguration  为了通过暂时使用MockDy
	dynamicConfiguration = config_center.MockDynamicConfiguration{}

	convert := util.Convert{}
	id := convert.GetIdFromDTO(override)

	path := getPath(id.(string))

	//TODO 改不出来了
	exitConfig, err := dynamicConfiguration.GetConfig(path)
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
		//TODO 更改  存在问题   SetConfig
		err := dynamicConfiguration.PublishConfig(path, "group", string(b))
		if err != nil {
			return
		}
	} else {
		panic(err)
	}
	//放在configCenter初始化的地方
	var regis registry.MockRegistry
	if override.Service != "" {
		result := convertDTOtoOldOverride(&override)
		for _, o := range result {
			//o.ToURL().AddParam(constant.CompatibleConfigKey, strconv.FormatBool(true))
			err := regis.Register(o.ToURL())
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

//	func GetIdFromDTO(dto *DynamicConfigDTO) string {
//		// Implement this function to get the ID from the DTO.
//	}
//
//	func GetPath(id string) string {
//		// Implement this function to get the path from the ID.
//	}
//
//	func ConvertDTOtoOldOverride(dto *DynamicConfigDTO) []Override {
//		// Implement this function to convert the DTO to an array of Override objects.
//	}
//
//	func contains(slice []string, s string) bool {
//		for _, e := range slice {
//			if e == s {
//				return true
//			}
//		}
//		return false
//	}
func getPath(key string) string {
	key = strings.Replace(key, "/", "*", -1)
	//ConfiguratorRuleSuffix
	return key + constant.ConfiguratorSuffix

}

//func (s *OverrideServiceImpl) FindOverride(id string) dto.DynamicConfigDTO {
//	return dto.DynamicConfigDTO{}
//}

//func (s *OverrideServiceImpl) UpdateOverride(update dto.DynamicConfigDTO) {
//	convert := util.Convert{}
//	id := convert.GetIdFromDTO(update)
//	path := getPath(id.(string))
//	exitConfig, err := dynamicConfiguration.GetConfig(path)
//	if exitConfig == "" {
//		log.Fatal("", err)
//		// throw exception
//	}
//	//exist,err := yaml.LoadYMLConfig(exitConfig)
//	//		if err!=nil{
//	//			log.Fatal("",err)
//	//		}
//	//		err = yaml.UnmarshalYML(exist,existOverride)
//	//		if err!=nil{
//	//			log.Fatal("",err)
//	//		}
//	//		if existOverride.GetConfigs()==nil{
//	//			for _, overrideConfig := range existOverride.Configs {
//	//				if HashSet.Contains(overrideConfig) {
//	//					configs = append(configs, overrideConfig)
//	//				}
//	//			}
//	//		}
//	var overrideDto dto.OverrideDTO
//	exist, err := yaml.LoadYMLConfig(exitConfig)
//	if err != nil {
//		log.Fatal("", err)
//	}
//
//	err = yaml.UnmarshalYML(exist, overrideDto)
//	if err != nil {
//		log.Fatal("", err)
//	}
//	old := util.CreateFromOverride(overrideDto)
//	configs := make([]store.OverrideConfig, 0)
//	if overrideDto.Configs != nil {
//		overrideConfigs := overrideDto.Configs
//		for _, config := range overrideConfigs {
//			if HashSet.Contains(config) {
//				configs = append(configs, config)
//			}
//		}
//	}
//	configs = append(configs, update.Configs...)
//	overrideDto.Configs = configs
//	overrideDto.Enabled = update.Enabled
//	dynamicConfiguration.set(path, YamlParser.dumpObject(overrideDTO))
//
//	// for 2.6
//	if update.Service != "" {
//		oldOverrides := convertDTOtoOldOverride(old)
//		updatedOverrides := convertDTOtoOldOverride(update)
//		for _, o := range oldOverrides {
//			registry.Unregister(o.ToUrl().AddParameter(Constants.COMPATIBLE_CONFIG, "true"))
//		}
//		for _, o := range updatedOverrides {
//			registry.Register(o.ToUrl().AddParameter(Constants.COMPATIBLE_CONFIG, "true"))
//		}
//	}
//
//}

//func (s *OverrideServiceImpl) DisableOverride(id string) {
//	if id == "" {
//		// throw exception
//	}
//	path := getPath(id)
//	if dynamicConfiguration.GetConfig(path) == "" {
//		// throw exception
//	}
//
//	config := dynamicConfiguration.GetConfig(path)
//	override := &dto.OverrideDTO{}
//	YamlParser.LoadObject(config, override)
//	old := OverrideUtils.CreateFromOverride(*override)
//	override.Enabled = false
//	dynamicConfiguration.SetConfig(path, YamlParser.DumpObject(override))
//
//	// for 2.6
//	if override.Scope == Constants.SERVICE {
//		overrides := convertDTOtoOldOverride(old)
//		for _, o := range overrides {
//			o.Enabled = true
//			registry.Unregister(o.ToUrl().AddParameter(Constants.COMPATIBLE_CONFIG, true))
//			o.Enabled = false
//			registry.Register(o.ToUrl().AddParameter(Constants.COMPATIBLE_CONFIG, true))
//		}
//	}
//}

//func (s *OverrideServiceImpl) FindOverride(id string) interface{} {
//	if id == "" {
//		// throw exception
//	}
//	path := getPath(id)
//	config := dynamicConfiguration.getConfig(path)
//	if config != "" {
//		overrideDTO := YamlParser.loadObject(config, OverrideDTO{})
//		dynamicConfigDTO := OverrideUtils.createFromOverride(overrideDTO)
//		if dynamicConfigDTO != nil {
//			dynamicConfigDTO.Id = id
//			if overrideDTO.Scope == Constants.SERVICE {
//				ConvertUtil.detachIdToService(id, dynamicConfigDTO)
//			}
//		}
//		return dynamicConfigDTO
//	}
//	return nil
//}

//func (s *OverrideServiceImpl) EnableOverride(id string) {
//	if id == "" {
//		// throw exception
//	}
//
//	path := getPath(id)
//	config := dynamicConfiguration.GetConfig(path)
//	if config == "" {
//		// throw exception
//	}
//
//	override := &OverrideDTO{}
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
