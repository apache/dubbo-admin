package config

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func TestGetSetConfig(t *testing.T) {
	err := SetConfig("test_key", "test_value")
	if err != nil {
		println("set config (test_key, test_value) error")
		panic(err)
	}
	config, _ := GetConfig("test_key")
	assert.Equal(t, "test_value", config)
	config, _ = GetConfig("not_exist_key")
	assert.Equal(t, "", config)

	err = SetConfigWithGroup("test_group", "test_key", "test_group_value")
	if err != nil {
		println("set config (test_group, test_key, test_group_value) error")
		panic(err)
	}
	config, _ = GetConfigWithGroup("test_group", "test_key")
	assert.Equal(t, "test_group_value", config)
	config, _ = GetConfigWithGroup("test_group", "not_exist_key")
	assert.Equal(t, "", config)

	_, err = GetConfig("")
	assert.Errorf(t, err, "key is empty")
	err = SetConfig("test_null", "")
	assert.Errorf(t, err, "key or value is empty")
}

func TestGetConifg(t *testing.T) {
	config, err := GetConfig("go1.tag-router")
	assert.NoError(t, err)
	println(config)
}

func TestSetConfig(t *testing.T) {
	err := SetConfig("test_key", "test_value")
	assert.NoError(t, err)
}

func setup() {
	address := "zookeeper://127.0.0.1:2181"
	c := newAddressConfig(address)
	factory, err := extension.GetConfigCenterFactory(c.getProtocol())
	if err != nil {
		panic(err)
	}
	url, err := c.toURL()
	if err != nil {
		panic(err)
	}

	ConfigCenter, err = factory.GetDynamicConfiguration(url)
	Group = url.GetParam(constant.GroupKey, constant.DefaultGroup)
}
