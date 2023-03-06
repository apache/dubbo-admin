package impl

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"encoding/json"
	"github.com/apache/dubbo-admin/pkg/admin/model/adapter"
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

// SaveOverride /**
var dynamicConfiguration config_center.DynamicConfiguration
var HashSet set.HashSet

func (s *OverrideServiceImpl) SaveOverride(override dto.DynamicConfigDTO) {
	HashSet = set.HashSet{
		Items: make(map[interface{}]struct{}),
	}
	// 怎么设置成动态的
	dynamicConfiguration = new(config_center.MockDynamicConfiguration)

	convert := util.Convert{}
	id := convert.GetIdFromDTO(override)

	path := getPath(id.(string))

	//TODO 改不出来了
	//exitConfig, err := dynamicConfiguration.GetConfigKeysByGroup(path)
	//if err != nil {
	//	log.Fatal("", err)
	//}
	configs := []store.OverrideConfig{}
	adapt, _ := adapter.NewDynamicConfigDTO2OverrideDTOAdapter(override)
	existOverride := adapt.OverrideDTO
	//parser := properties.Parser()

	if reflect.ValueOf(existOverride).IsNil() {
		//if err, _ := parser.Marshal([]byte(exitConfig), existOverride); err != nil {
		//	panic(err)
		//}
		//TODO 不会写
		//if err, _ := parser.Marshal((exitConfig.Items).(map[string]interface{})); err != nil {
		//	panic(err)
		//}

		if existOverride.GetConfigs != nil {
			for _, overrideConfig := range existOverride.Configs {
				//TODO 这个是全局hash吗？
				if HashSet.Contains(overrideConfig) {
					configs = append(configs, overrideConfig)
				}
			}
		}
	}
	configs = append(configs, override.Configs...)
	existOverride.Enabled = override.Enabled
	existOverride.Configs = configs
	if b, err := json.Marshal(existOverride); err == nil {
		//TODO 更改
		err := dynamicConfiguration.PublishConfig(path, "group", string(b))
		if err != nil {
			return
		}
	} else {
		panic(err)
	}

	// for 2.6
	//if override.Service != "" {
	//	result := ConvertDTOtoOldOverride(override)
	//	for _, o := range result {
	//		url := o.ToUrl().AddParameter(Constants.COMPATIBLE_CONFIG, true)
	//		registry.Register(url)
	//	}
	//}
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

func (s *OverrideServiceImpl) UpdateOverride(overrideDTO dto.DynamicConfigDTO) {

}

func (s *OverrideServiceImpl) DisableOverride(id string) {

}

func (s *OverrideServiceImpl) FindOverride(id string) interface{} {
	return nil
}

func (s *OverrideServiceImpl) EnableOverride(id string) {

}

func (s *OverrideServiceImpl) DeleteOverride(id string) {

}

//func (s *OverrideServiceImpl) convertDTOtoOldOverride(overrideDTO dto.DynamicConfigDTO) []domain.Override {
//	result := []domain.Override{}
//	configs := overrideDTO.Configs
//	for _, config := range configs {
//		if HashSet.Contains(config.Type)  {
//			continue
//		}
//		apps := config.Applications
//		addresses := config.Addresses
//		for _, address := range addresses {
//			if apps != nil && len(apps) > 0 {
//				for _, app := range apps {
//					override := Override{
//						Service:  overrideDTO.Service,
//						Address:  address,
//						Enabled:  overrideDTO.Enabled,
//						Application: app,
//					}
//					overrideDTOToParams(&override, config)
//					result = append(result, override)
//				}
//			} else {
//				override := Override{
//					Service:  overrideDTO.Service,
//					Address:  address,
//					Enabled:  overrideDTO.Enabled,
//				}
//				overrideDTOToParams(&override, config)
//				result = append(result, override)
//			}
//		}
//	}
//	return result
//}
//
//
//
//const (
//	ConstantsCONFIGS = "CONFIGS"
//)
//
//func contains(list []string, elem string) bool {
//	for _, e := range list {
//		if e == elem {
//			return true
//		}
//	}
//	return false
//}
//
//func overrideDTOToParams(override *Override, config OverrideConfig) {
//	override.Params = config.Params
//}
//
