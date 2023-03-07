package dto

import (
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"hash/fnv"
	"reflect"
	"strings"
)

type ServiceDTO struct {
	Service        string
	AppName        string
	Group          string
	Version        string
	RegistrySource model.RegistrySource
}

//func (s *ServiceDTO) GetAppName() string {
//	return s.AppName
//}

func (s ServiceDTO) CompareTo(o ServiceDTO) int {
	result := strings.TrimSpace(s.AppName) == strings.TrimSpace(o.AppName)
	if result == true {
		result = strings.TrimSpace(s.Service) == strings.TrimSpace(o.Service)
		if result == true {
			result = strings.TrimSpace(s.Group) == strings.TrimSpace(o.Group)
		}
		if result == true {
			result = strings.TrimSpace(s.Version) == strings.TrimSpace(o.Version)
		}
		if result == true {
			result = s.RegistrySource == o.RegistrySource
		}
	}
	if result == true {
		return 0
	}
	return 1
}

func (s ServiceDTO) Equals(o interface{}) bool {
	if s == o {
		return true
	}
	if o == nil || reflect.TypeOf(s) != reflect.TypeOf(o) {
		return false
	}
	that := o.(ServiceDTO)
	return s.Service == that.Service && s.AppName == that.AppName && s.Group == that.Group && s.Version == that.Version && s.RegistrySource == that.RegistrySource
}

func (s ServiceDTO) HashCode() int {
	h := fnv.New32a()
	h.Write([]byte(s.Service))
	h.Write([]byte(s.AppName))
	h.Write([]byte(s.Group))
	h.Write([]byte(s.Version))
	//h.Write([]byte(s.RegistrySource))
	return int(h.Sum32())
}
