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
	"net/url"
	"regexp"
	"sync"
	"testing"

	set "github.com/dubbogo/gost/container/set"

	"github.com/stretchr/testify/assert"

	"github.com/apache/dubbo-admin/pkg/admin/model/util"

	"github.com/apache/dubbo-admin/pkg/admin/constant"

	"github.com/apache/dubbo-admin/pkg/admin/cache"

	"dubbo.apache.org/dubbo-go/v3/common"

	"github.com/apache/dubbo-admin/pkg/admin/model"
)

var testProvider *model.Provider

func initCacheMock() {
	service := &sync.Map{}
	queryParams := url.Values{
		constant.ApplicationKey: {"test"},
	}
	testURL, _ := common.NewURL(common.GetLocalIp()+":0",
		common.WithProtocol(constant.AdminProtocol),
		common.WithParams(queryParams),
		common.WithLocation(common.GetLocalIp()+":0"),
	)
	service.Store("test", map[string]*common.URL{
		"test": testURL,
	})
	cache.InterfaceRegistryCache.Store(constant.ProvidersCategory, service)
	testProvider = util.URL2Provider("test", testURL)
}

func TestProviderServiceImpl_FindServices(t *testing.T) {
	initCacheMock()
	defer cache.InterfaceRegistryCache.Delete(constant.ProvidersCategory)
	tests := []struct {
		name    string
		want    *set.HashSet
		wantErr bool
	}{
		{
			name:    "Test",
			want:    set.NewSet("test"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProviderServiceImpl{}
			got, err := p.FindServices()
			if (err != nil) != tt.wantErr {
				t.Errorf("FindServices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProviderServiceImpl_FindApplications(t *testing.T) {
	initCacheMock()
	defer cache.InterfaceRegistryCache.Delete(constant.ProvidersCategory)
	tests := []struct {
		name    string
		want    *set.HashSet
		wantErr bool
	}{
		{
			name:    "Test",
			want:    set.NewSet("test"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProviderServiceImpl{}
			got, err := p.FindApplications()
			if (err != nil) != tt.wantErr {
				t.Errorf("FindApplications() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProviderServiceImpl_findAddresses(t *testing.T) {
	initCacheMock()
	defer cache.InterfaceRegistryCache.Delete(constant.ProvidersCategory)
	tests := []struct {
		name    string
		want    *set.HashSet
		wantErr bool
	}{
		{
			name:    "Test",
			want:    set.NewSet(common.GetLocalIp() + ":0"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProviderServiceImpl{}
			got, err := p.findAddresses()
			if (err != nil) != tt.wantErr {
				t.Errorf("findAddresses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProviderServiceImpl_FindByService(t *testing.T) {
	initCacheMock()
	defer cache.InterfaceRegistryCache.Delete(constant.ProvidersCategory)
	type args struct {
		providerService string
	}
	tests := []struct {
		name    string
		args    args
		want    []*model.Provider
		wantErr bool
	}{
		{
			name: "Test",
			args: args{
				providerService: "test",
			},
			want:    []*model.Provider{testProvider},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProviderServiceImpl{}
			got, err := p.FindByService(tt.args.providerService)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProviderServiceImpl_findByAddress(t *testing.T) {
	initCacheMock()
	defer cache.InterfaceRegistryCache.Delete(constant.ProvidersCategory)
	type args struct {
		providerAddress string
	}
	tests := []struct {
		name    string
		args    args
		want    []*model.Provider
		wantErr bool
	}{
		{
			name: "Test",
			args: args{
				providerAddress: common.GetLocalIp() + ":0",
			},
			want:    []*model.Provider{testProvider},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProviderServiceImpl{}
			got, err := p.findByAddress(tt.args.providerAddress)
			if (err != nil) != tt.wantErr {
				t.Errorf("findByAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProviderServiceImpl_findByApplication(t *testing.T) {
	initCacheMock()
	defer cache.InterfaceRegistryCache.Delete(constant.ProvidersCategory)
	type args struct {
		providerApplication string
	}
	tests := []struct {
		name    string
		args    args
		want    []*model.Provider
		wantErr bool
	}{
		{
			name: "Test",
			args: args{
				providerApplication: "test",
			},
			want:    []*model.Provider{testProvider},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProviderServiceImpl{}
			got, err := p.findByApplication(tt.args.providerApplication)
			if (err != nil) != tt.wantErr {
				t.Errorf("findByApplication() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProviderServiceImpl_FindService(t *testing.T) {
	type args struct {
		pattern string
		filter  string
	}
	tests := []struct {
		name    string
		args    args
		want    []*model.ServiceDTO
		wantErr bool
	}{
		{
			name: "Test",
			args: args{
				pattern: "ip",
				filter:  "test",
			},
			want:    make([]*model.ServiceDTO, 0),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProviderServiceImpl{}
			got, err := p.FindService(tt.args.pattern, tt.args.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReg(t *testing.T) {
	reg, _ := regexp.Compile(".*DemoService*")
	match := reg.MatchString("org.apache.dubbo.springboot.demo.DemoService")
	if match {
		fmt.Print("Matched!")
	}
}
