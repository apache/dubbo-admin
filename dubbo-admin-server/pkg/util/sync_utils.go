package util

import (
	"admin/pkg/constant"
	"admin/pkg/model"
	"dubbo.apache.org/dubbo-go/v3/common"
)

func Url2Provider(id string, url *common.URL) *model.Provider {
	if url == nil {
		return nil
	}

	return &model.Provider{
		Entity:         model.Entity{Hash: id},
		Service:        url.ServiceKey(),
		Address:        url.Location,
		Application:    url.GetParam(constant.ApplicationKey, ""),
		Url:            url.Key(),
		Parameters:     url.String(),
		Dynamic:        url.GetParamBool(constant.DynamicKey, true),
		Enabled:        url.GetParamBool(constant.EnabledKey, true),
		Serialization:  url.GetParam(constant.SerializationKey, "hessian2"),
		Timeout:        url.GetParamInt(constant.TimeoutKey, constant.DefaultTimeout),
		Weight:         url.GetParamInt(constant.WeightKey, constant.DefaultWeight),
		Username:       url.GetParam(constant.OwnerKey, ""),
		RegistrySource: model.INTERFACE,
	}
}
