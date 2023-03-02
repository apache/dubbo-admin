package model

import "time"

type Provider struct {
	Entity
	Service        string
	Url            string
	Parameters     string
	Address        string
	Registry       string
	Dynamic        bool
	Enabled        bool
	Timeout        int64
	Serialization  string
	Weight         int64
	Application    string
	Username       string
	Expired        time.Duration
	Alived         int64
	RegistrySource RegistrySource
}
