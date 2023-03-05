package config

import (
	"net/url"
)

type GovernanceConfiguration interface {
	Init()
	SetURL(url *url.URL)
	GetURL() *url.URL
	SetConfig(key, value string) bool
	GetConfig(key string) string
	DeleteConfig(key string) bool
	SetGroupConfig(group, key, value string) bool
	GetGroupConfig(group, key string) string
	DeleteGroupConfig(group, key string) bool
	GetPath(key string) string
	GetGroupPath(group, key string) string
}

//type governanceConfig struct {
//	url         *url.URL
//	configs     map[string]string
//	groupConfig map[string]map[string]string
//	mutex       sync.RWMutex
//}
