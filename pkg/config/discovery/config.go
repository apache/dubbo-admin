package discovery

import "github.com/apache/dubbo-admin/pkg/config"

type Type string

const (
	zookeeper Type = "zookeeper"
	nacos     Type = "nacos"
)

// Config defines Discovery Engine configuration
type Config struct {
	config.BaseConfig
	Name    string `json:"name"`
	Type    Type   `json:"type"`
	Address AddressConfig
}

// AddressConfig defines Discovery Engine address
type AddressConfig struct {
	Registry       string `json:"registry"`
	ConfigCenter   string `json:"configCenter"`
	MetadataReport string `json:"metadataReport"`
}

func DefaultDiscoveryEnginConfig() *Config {
	return &Config{
		Name: "localhost",
		Type: nacos,
		Address: AddressConfig{
			Registry:       "nacos://127.0.0.1:8848?username=nacos&password=nacos",
			ConfigCenter:   "nacos://127.0.0.1:8848?username=nacos&password=nacos",
			MetadataReport: "nacos://127.0.0.1:8848?username=nacos&password=nacos",
		},
	}
}
