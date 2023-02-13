package util

import (
	"strings"
)

func GetInterface(service string) string {
	if len(service) > 0 {
		index := strings.Index(service, "/")
		if index >= 0 {
			service = service[index+1:]
		}
		index = strings.LastIndex(service, ":")
		if index >= 0 {
			service = service[0:index]
		}
	}
	return service
}

func GetGroup(service string) string {
	if len(service) > 0 {
		index := strings.Index(service, "/")
		if index >= 0 {
			return service[0:index]
		}
	}
	return ""
}

func GetVersion(service string) string {
	if len(service) > 0 {
		index := strings.LastIndex(service, ":")
		if index >= 0 {
			return service[index+1:]
		}
	}
	return ""
}
