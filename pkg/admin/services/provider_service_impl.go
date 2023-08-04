// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	set "github.com/dubbogo/gost/container/set"

	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/apache/dubbo-admin/pkg/admin/cache"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/model/util"
)

type ProviderServiceImpl struct{}

// FindServices finds all services
func (p *ProviderServiceImpl) FindServices() (*set.HashSet, error) {
	services := set.NewSet()
	servicesAny, ok := cache.InterfaceRegistryCache.Load(constant.ProvidersCategory)
	if !ok {
		return services, nil
	}
	servicesMap, ok := servicesAny.(*sync.Map)
	if !ok {
		return services, fmt.Errorf("servicesMap type not *sync.Map")
	}

	servicesMap.Range(func(key, value any) bool {
		services.Add(key.(string))
		return true
	})
	return services, nil
}

// FindApplications finds all applications
func (p *ProviderServiceImpl) FindApplications() (*set.HashSet, error) {
	var (
		applications = set.NewSet()
		err          error
	)
	providersAny, ok := cache.InterfaceRegistryCache.Load(constant.ProvidersCategory)
	if !ok {
		return applications, nil
	}
	err = extractApplications(providersAny, applications)
	if err != nil {
		return applications, err
	}

	consumersAny, ok := cache.InterfaceRegistryCache.Load(constant.ConsumersCategory)
	if !ok {
		return applications, nil
	}
	err = extractApplications(consumersAny, applications)
	if err != nil {
		return applications, err
	}
	return applications, err
}

func extractApplications(servicesAny any, applications *set.HashSet) error {
	servicesMap, ok := servicesAny.(*sync.Map)
	if !ok {
		return fmt.Errorf("servicesMap type not *sync.Map")
	}

	var err error
	servicesMap.Range(func(key, value any) bool {
		service, ok := value.(map[string]*common.URL)
		if !ok {
			err = fmt.Errorf("service type not map[string]*common.URL")
			return false
		}
		for _, url := range service {
			app := url.GetParam(constant.ApplicationKey, "")
			if app != "" {
				applications.Add(app)
			}
		}
		return true
	})
	return err
}

// findAddresses finds all addresses
func (p *ProviderServiceImpl) findAddresses() (*set.HashSet, error) {
	var (
		addresses = set.NewSet()
		err       error
	)
	servicesAny, ok := cache.InterfaceRegistryCache.Load(constant.ProvidersCategory)
	if !ok {
		return addresses, nil
	}
	err = extractAddresses(servicesAny, addresses)
	if err != nil {
		return addresses, err
	}

	consumersAny, ok := cache.InterfaceRegistryCache.Load(constant.ConsumersCategory)
	if !ok {
		return addresses, nil
	}
	err = extractAddresses(consumersAny, addresses)
	if err != nil {
		return addresses, err
	}

	return addresses, err
}

func extractAddresses(servicesAny any, addresses *set.HashSet) error {
	servicesMap, ok := servicesAny.(*sync.Map)
	if !ok {
		return fmt.Errorf("servicesMap type not *sync.Map")
	}

	var err error
	servicesMap.Range(func(key, value any) bool {
		service, ok := value.(map[string]*common.URL)
		if !ok {
			err = fmt.Errorf("service type not map[string]*common.URL")
			return false
		}
		for _, url := range service {
			loc := url.Location
			if loc != "" {
				addresses.Add(loc)
			}
		}
		return true
	})
	return err
}

// FindVersions finds all versions
func (p *ProviderServiceImpl) FindVersions() (*set.HashSet, error) {
	var (
		versions = set.NewSet()
		err      error
	)
	servicesAny, ok := cache.InterfaceRegistryCache.Load(constant.ProvidersCategory)
	if !ok {
		return versions, nil
	}

	err = extractVersions(servicesAny, versions)
	if err != nil {
		return versions, err
	}

	return versions, err
}

func extractVersions(servicesAny any, versions *set.HashSet) error {
	servicesMap, ok := servicesAny.(*sync.Map)
	if !ok {
		return fmt.Errorf("servicesMap type not *sync.Map")
	}

	var err error
	servicesMap.Range(func(key, value any) bool {
		service, ok := value.(map[string]*common.URL)
		if !ok {
			err = fmt.Errorf("service type not map[string]*common.URL")
			return false
		}
		for _, url := range service {
			release := url.GetParam("release", "")
			if release == "" {
				release = url.GetParam("revision", "")
			}
			if release != "" {
				versions.Add(release)
			}
		}
		return true
	})
	return err
}

// FindProtocols finds all protocols
func (p *ProviderServiceImpl) FindProtocols() (*set.HashSet, error) {
	var (
		protocols = set.NewSet()
		err       error
	)
	servicesAny, ok := cache.InterfaceRegistryCache.Load(constant.ProvidersCategory)
	if !ok {
		return protocols, nil
	}

	err = extractProtocols(servicesAny, protocols)
	if err != nil {
		return protocols, err
	}

	return protocols, err
}

func extractProtocols(servicesAny any, protocols *set.HashSet) error {
	servicesMap, ok := servicesAny.(*sync.Map)
	if !ok {
		return fmt.Errorf("servicesMap type not *sync.Map")
	}

	var err error
	servicesMap.Range(func(key, value any) bool {
		service, ok := value.(map[string]*common.URL)
		if !ok {
			err = fmt.Errorf("service type not map[string]*common.URL")
			return false
		}
		for _, url := range service {
			proto := url.Protocol
			if proto != "" && proto != "consumer" {
				protocols.Add(proto)
			}
		}
		return true
	})
	return err
}

// FindByService finds providers by service name and returns a list of providers
func (p *ProviderServiceImpl) FindByService(providerService string) ([]*model.Provider, error) {
	filter := make(map[string]string)
	filter[constant.CategoryKey] = constant.ProvidersCategory
	filter[util.ServiceFilterKey] = providerService
	servicesMap, err := util.FilterFromCategory(filter)
	if err != nil {
		return nil, err
	}
	return util.URL2ProviderList(servicesMap), nil
}

// findByAddress finds providers by address and returns a list of providers
func (p *ProviderServiceImpl) findByAddress(providerAddress string) ([]*model.Provider, error) {
	filter := make(map[string]string)
	filter[constant.CategoryKey] = constant.ProvidersCategory
	filter[util.AddressFilterKey] = providerAddress
	servicesMap, err := util.FilterFromCategory(filter)
	if err != nil {
		return nil, err
	}
	return util.URL2ProviderList(servicesMap), nil
}

// findByApplication finds providers by application and returns a list of providers
func (p *ProviderServiceImpl) findByApplication(providerApplication string) ([]*model.Provider, error) {
	filter := make(map[string]string)
	filter[constant.CategoryKey] = constant.ProvidersCategory
	filter[constant.ApplicationKey] = providerApplication
	servicesMap, err := util.FilterFromCategory(filter)
	if err != nil {
		return nil, err
	}
	return util.URL2ProviderList(servicesMap), nil
}

// FindService by patterns and filters, patterns support IP, service and application.
func (p *ProviderServiceImpl) FindService(pattern string, filter string) ([]*model.ServiceDTO, error) {
	var (
		providers []*model.Provider
		reg       *regexp.Regexp
		err       error
	)
	result := make([]*model.Provider, 0)
	if !strings.Contains(filter, constant.AnyValue) && !strings.Contains(filter, constant.InterrogationPoint) {
		if pattern == constant.IP {
			providers, err = p.findByAddress(filter)
			if err != nil {
				return nil, err
			}
		} else if pattern == constant.Service {
			providers, err = p.FindByService(filter)
			if err != nil {
				return nil, err
			}
		} else if pattern == constant.ApplicationKey {
			providers, err = p.findByApplication(filter)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unsupport the pattern: %s", pattern)
		}
		result = providers
	} else {
		var candidates *set.HashSet
		if pattern == constant.IP {
			candidates, err = p.findAddresses()
			if err != nil {
				return nil, err
			}
		} else if pattern == constant.Service {
			candidates, err = p.FindServices()
			if err != nil {
				return nil, err
			}
		} else if pattern == constant.ApplicationKey {
			candidates, err = p.FindApplications()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unsupport the pattern: %s", pattern)
		}

		filter = strings.ReplaceAll(filter, constant.PunctuationPoint, "\\.")
		if hasPrefixOrSuffix(filter) {
			filter = strings.ReplaceAll(filter, constant.AnyValue, constant.PunctuationPoint+constant.AnyValue)
		}
		reg, err = regexp.Compile(filter)
		if err != nil {
			return nil, err
		}
		items := candidates.Values()
		for _, candidateAny := range items {
			candidate := candidateAny.(string)
			if reg.MatchString(candidate) {
				if pattern == constant.IP {
					providers, err = p.findByAddress(candidate)
					if err != nil {
						return nil, err
					}
				} else if pattern == constant.Service {
					providers, err = p.FindByService(candidate)
					if err != nil {
						return nil, err
					}
				} else if pattern == constant.ApplicationKey {
					providers, err = p.findByApplication(candidate)
					if err != nil {
						return nil, err
					}
				}
				result = append(result, providers...)
			}
		}
	}

	return util.Providers2DTO(result), nil
}

func hasPrefixOrSuffix(filter string) bool {
	return strings.HasPrefix(filter, constant.AnyValue) || strings.HasPrefix(filter, constant.InterrogationPoint) ||
		strings.HasPrefix(filter, constant.PlusSigns) || strings.HasSuffix(filter, constant.AnyValue) || strings.HasSuffix(filter, constant.InterrogationPoint) ||
		strings.HasSuffix(filter, constant.PlusSigns)
}
