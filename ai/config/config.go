package config

import (
	"gopkg.in/yaml.v3"
)

// Config component configuration structure
type Config struct {
	Type string    `yaml:"type"`
	Spec yaml.Node `yaml:"spec"`
}

func (c *Config) GetType() string {
	return c.Type
}

func (c *Config) GetSpec() *yaml.Node {
	return &c.Spec
}
