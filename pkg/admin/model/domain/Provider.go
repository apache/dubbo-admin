package domain

import "time"

type Provider struct {
	Entity
	Service        string
	URL            string
	Parameters     string
	Address        string
	Registry       string
	Dynamic        bool
	Enabled        bool
	Timeout        int
	Serialization  string
	Weight         int
	Application    string
	Username       string
	Expired        time.Time
	Alived         int64
	Override       *Override
	Overrides      []*Override
	RegistrySource *RegistrySource
}

func NewProvider() *Provider {
	return &Provider{}
}
