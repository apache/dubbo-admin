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

	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/apache/dubbo-admin/pkg/admin/cache"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/util"
)

type ProviderServiceImpl struct{}

// FindServices finds all services
func (p *ProviderServiceImpl) FindServices() ([]string, error) {
	var services []string
	servicesAny, ok := cache.InterfaceRegistryCache.Load(constant.ProvidersCategory)
	if !ok {
		return nil, nil
	}
	servicesMap, ok := servicesAny.(*sync.Map)
	if !ok {
		return nil, fmt.Errorf("servicesMap type not *sync.Map")
	}

	servicesMap.Range(func(key, value any) bool {
		services = append(services, key.(string))
		return true
	})
	return services, nil
}

// FindApplications finds all applications
func (p *ProviderServiceImpl) FindApplications() ([]string, error) {
	var (
		applications []string
		err          error
	)
	servicesAny, ok := cache.InterfaceRegistryCache.Load(constant.ProvidersCategory)
	if !ok {
		return nil, nil
	}
	servicesMap, ok := servicesAny.(*sync.Map)
	if !ok {
		return nil, fmt.Errorf("servicesMap type not *sync.Map")
	}

	servicesMap.Range(func(key, value any) bool {
		service, ok := value.(map[string]*common.URL)
		if !ok {
			err = fmt.Errorf("service type not map[string]*common.URL")
			return false
		}
		for _, url := range service {
			app := url.GetParam(constant.ApplicationKey, "")
			if app != "" {
				applications = append(applications, app)
			}
		}
		return true
	})
	return applications, err
}

// findAddresses finds all addresses
func (p *ProviderServiceImpl) findAddresses() ([]string, error) {
	var (
		addresses []string
		err       error
	)
	servicesAny, ok := cache.InterfaceRegistryCache.Load(constant.ProvidersCategory)
	if !ok {
		return nil, nil
	}
	servicesMap, ok := servicesAny.(*sync.Map)
	if !ok {
		return nil, fmt.Errorf("servicesMap type not *sync.Map")
	}

	servicesMap.Range(func(key, value any) bool {
		service, ok := value.(map[string]*common.URL)
		if !ok {
			err = fmt.Errorf("service type not map[string]*common.URL")
			return false
		}
		for _, url := range service {
			loc := url.Location
			if loc != "" {
				addresses = append(addresses, loc)
			}
		}
		return true
	})
	return addresses, err
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

// FindByApplication finds providers by address and returns a list of providers
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
func (p *ProviderServiceImpl) FindService(pattern string, filter string) ([]*model.Provider, error) {
	var (
		providers []*model.Provider
		reg       *regexp.Regexp
		err       error
	)
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
	} else {
		var candidates []string
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

		filter = strings.ToLower(filter)
		if strings.HasPrefix(filter, constant.AnyValue) || strings.HasPrefix(filter, constant.InterrogationPoint) ||
			strings.HasPrefix(filter, constant.PlusSigns) {
			filter = constant.PunctuationPoint + filter
		}
		reg, err = regexp.Compile(filter)
		if err != nil {
			return nil, err
		}
		for _, candidate := range candidates {
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
			}
		}
	}

	return providers, nil
}
