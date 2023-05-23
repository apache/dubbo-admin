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

package util

import (
	"reflect"
	"sync"
	"testing"

	"github.com/apache/dubbo-admin/pkg/admin/constant"

	"dubbo.apache.org/dubbo-go/v3/common"
)

func TestFilterFromCategory(t *testing.T) {
	type args struct {
		filter map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]*common.URL
		wantErr bool
	}{
		{
			name: "RightTest",
			args: args{
				filter: map[string]string{
					constant.CategoryKey: constant.ProvidersCategory,
				},
			},
			want:    map[string]*common.URL{},
			wantErr: false,
		},
		{
			name: "WrongTest",
			args: args{
				filter: map[string]string{},
			},
			want:    map[string]*common.URL{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FilterFromCategory(tt.args.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("FilterFromCategory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_filterFromService(t *testing.T) {
	type args struct {
		servicesMap *sync.Map
		filter      map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]*common.URL
		wantErr bool
	}{
		{
			name: "RightTest",
			args: args{
				servicesMap: &sync.Map{},
				filter: map[string]string{
					ServiceFilterKey: "test",
				},
			},
			want: map[string]*common.URL{
				"test": {},
			},
			wantErr: false,
		},
		{
			name: "WrongTest",
			args: args{
				servicesMap: &sync.Map{},
				filter: map[string]string{
					ServiceFilterKey: "test",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	tests[0].args.servicesMap.Store("test", map[string]*common.URL{
		"test": {},
	})
	tests[1].args.servicesMap.Store("test", map[string]string{
		"test": "string",
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterFromService(tt.args.servicesMap, tt.args.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("filterFromService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterFromService() got = %v, want %v", got, tt.want)
			}
		})
	}
}
