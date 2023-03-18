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
	"reflect"
	"testing"

	"github.com/apache/dubbo-admin/pkg/admin/model"
)

func TestProviderServiceImpl_FindServices(t *testing.T) {
	tests := []struct {
		name    string
		want    []string
		wantErr bool
	}{
		{
			name:    "Test",
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProviderServiceImpl{}
			_, err := p.FindServices()
			if (err != nil) != tt.wantErr {
				t.Errorf("FindServices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestProviderServiceImpl_FindApplications(t *testing.T) {
	tests := []struct {
		name    string
		want    []string
		wantErr bool
	}{
		{
			name:    "Test",
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProviderServiceImpl{}
			_, err := p.FindApplications()
			if (err != nil) != tt.wantErr {
				t.Errorf("FindApplications() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestProviderServiceImpl_findAddresses(t *testing.T) {
	tests := []struct {
		name    string
		want    []string
		wantErr bool
	}{
		{
			name:    "Test",
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProviderServiceImpl{}
			_, err := p.findAddresses()
			if (err != nil) != tt.wantErr {
				t.Errorf("findAddresses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestProviderServiceImpl_FindByService(t *testing.T) {
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
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProviderServiceImpl{}
			_, err := p.FindByService(tt.args.providerService)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestProviderServiceImpl_findByAddress(t *testing.T) {
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
				providerAddress: "test",
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProviderServiceImpl{}
			_, err := p.findByAddress(tt.args.providerAddress)
			if (err != nil) != tt.wantErr {
				t.Errorf("findByAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestProviderServiceImpl_findByApplication(t *testing.T) {
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
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProviderServiceImpl{}
			_, err := p.findByApplication(tt.args.providerApplication)
			if (err != nil) != tt.wantErr {
				t.Errorf("findByApplication() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
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
		want    []*model.Provider
		wantErr bool
	}{
		{
			name: "Test",
			args: args{
				pattern: "ip",
				filter:  "test",
			},
			want:    nil,
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindService() got = %v, want %v", got, tt.want)
			}
		})
	}
}
