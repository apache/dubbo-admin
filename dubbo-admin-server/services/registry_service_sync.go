package services

import (
	"admin/constant"
	"dubbo.apache.org/dubbo-go/v3/common"
	"net/url"
)

var SUBSCRIBE *url.URL

func init() {
	queryParams := url.Values{
		constant.InterfaceKey:  {constant.AnyValue},
		constant.GroupKey:      {constant.AnyValue},
		constant.VersionKey:    {constant.AnyValue},
		constant.ClassifierKey: {constant.AnyValue},
		constant.CategoryKey: {constant.ProvidersCategory +
			"," + constant.ConsumersCategory +
			"," + constant.RoutersCategory +
			"," + constant.ConfiguratorsCategory},
		constant.EnabledKey: {constant.AnyValue},
		constant.CheckKey:   {"false"},
	}
	SUBSCRIBE = &url.URL{
		Scheme:   constant.AdminProtocol,
		Host:     common.GetLocalIp() + ":0",
		Path:     "",
		RawQuery: queryParams.Encode(),
	}
}

