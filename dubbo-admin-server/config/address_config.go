package config

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"net/url"
)

type AddressConfig struct {
	Address string `yaml:"address"`
	url     *url.URL
}

func (c *AddressConfig) getProtocol() string {
	return c.url.Scheme
}

func (c *AddressConfig) getAddress() string {
	return c.url.Host
}

func (c *AddressConfig) GetUrlMap() url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.ConfigNamespaceKey, c.param("namespace", ""))
	urlMap.Set(constant.ConfigGroupKey, c.param("group", ""))
	return urlMap
}

func (c *AddressConfig) param(key string, defaultValue string) string {
	param := c.url.Query().Get(key)
	if len(param) > 0 {
		return param
	}
	return defaultValue
}

func (c *AddressConfig) toURL() (*common.URL, error) {
	return common.NewURL(c.getAddress(),
		common.WithProtocol(c.getProtocol()),
		common.WithParams(c.GetUrlMap()),
		common.WithUsername(c.param("username", "")),
		common.WithPassword(c.param("password", "")),
	)
}
