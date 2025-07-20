package engine

import "github.com/apache/dubbo-admin/pkg/config"

type Type string

const (
	VM         Type = "vm"
	Kubernetes Type = "kubernetes"
)

type Config struct {
	config.BaseConfig
	Name string `json:"name"`
	Type Type   `json:"type"`
}

// AddressConfig defines Discovery Engine address
type AddressConfig struct {
	Registry       string `json:"registry"`
	ConfigCenter   string `json:"configCenter"`
	MetadataReport string `json:"metadataReport"`
}

func DefaultResourceEngineConfig() *Config {
	return &Config{
		Name: "default",
		Type: VM,
	}
}
