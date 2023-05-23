/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package traffic

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/config/mock_config"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/golang/mock/gomock"

	"sigs.k8s.io/yaml"
)

func TestCreateTimeout(t *testing.T) {
	var capturedRule string

	ctrl := gomock.NewController(t)
	mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().SetConfig(gomock.Any(), gomock.Any()).Do(func(key, rule string) {
		capturedRule = rule
	})
	mockGovernanceConfig.EXPECT().GetConfig(gomock.Any()).Return("", nil)
	config.Governance = mockGovernanceConfig

	tests := []struct {
		name    string
		args    *model.Timeout
		want    string
		wantErr bool
	}{
		{
			name: "create_timeout",
			args: &model.Timeout{
				Service: "DemoService",
				Group:   "",
				Version: "",
				Timeout: 5000,
			},
			want:    "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      timeout: 5000",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeoutSvc := &TimeoutService{}

			if err := timeoutSvc.CreateOrUpdate(tt.args); err != nil && !tt.wantErr {
				t.Errorf("TimeoutService.CreateOrUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}

			actualJson, _ := yaml.YAMLToJSON([]byte(capturedRule))
			wantedJson, _ := yaml.YAMLToJSON([]byte(tt.want))
			if !jsonpatch.Equal(actualJson, wantedJson) {
				t.Errorf("TimeoutService.CreateOrUpdate() error \n, expected:\n %v \n, got:\n  %v \n", tt.want, capturedRule)
			}
		})
	}
}

func TestUpdateTimeout(t *testing.T) {
	var capturedRule string

	tests := []struct {
		name         string
		args         *model.Timeout
		want         string
		existingRule string
		wantErr      bool
	}{
		{
			name: "update_timeout_multi_configs",
			args: &model.Timeout{
				Service: "DemoService",
				Group:   "",
				Version: "",
				Timeout: 6000,
			},
			existingRule: "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      timeout: 5000\n      test: \"value\"\n  - side: provider\n    enabled: true\n    parameters:\n      accesslog: true",
			want:         "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      timeout: 6000\n      test: \"value\"\n  - side: provider\n    enabled: true\n    parameters:\n      accesslog: true",
			wantErr:      false,
		},
		{
			name: "update_timeout_single_config_single_item",
			args: &model.Timeout{
				Service: "DemoService",
				Group:   "",
				Version: "",
				Timeout: 6000,
			},
			existingRule: "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      timeout: 5000",
			want:         "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      timeout: 6000",
			wantErr:      false,
		},
		{
			name: "update_timeout_single_config_multi_items",
			args: &model.Timeout{
				Service: "DemoService",
				Group:   "",
				Version: "",
				Timeout: 6000,
			},
			existingRule: "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      timeout: 5000\n      test: \"value\"",
			want:         "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      timeout: 6000\n      test: \"value\"",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeoutSvc := &TimeoutService{}

			ctrl := gomock.NewController(t)
			mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
			mockGovernanceConfig.EXPECT().SetConfig(gomock.Any(), gomock.Any()).Do(func(key, rule string) {
				capturedRule = rule
			})
			mockGovernanceConfig.EXPECT().GetConfig(gomock.Any()).Return(tt.existingRule, nil)
			config.Governance = mockGovernanceConfig

			if err := timeoutSvc.CreateOrUpdate(tt.args); err != nil && !tt.wantErr {
				t.Errorf("TimeoutService.CreateOrUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}

			actualJson, _ := yaml.YAMLToJSON([]byte(capturedRule))
			wantedJson, _ := yaml.YAMLToJSON([]byte(tt.want))
			if !jsonpatch.Equal(actualJson, wantedJson) {
				t.Errorf("TimeoutService.CreateOrUpdate() error \n, expected:\n %v \n, got:\n  %v \n", tt.want, capturedRule)
			}
		})
	}
}

func TestDeleteTimeout(t *testing.T) {
	var capturedRule string

	tests := []struct {
		name         string
		args         *model.Timeout
		want         string
		existingRule string
		wantDelete   bool
		wantErr      bool
	}{
		{
			name: "delete_timeout_multi_configs",
			args: &model.Timeout{
				Service: "DemoService",
				Group:   "",
				Version: "",
				Timeout: 5000,
			},
			existingRule: "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      timeout: 5000\n      test: \"value\"\n  - side: provider\n    enabled: true\n    parameters:\n      accesslog: true",
			want:         "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      test: \"value\"\n  - side: provider\n    enabled: true\n    parameters:\n      accesslog: true",
			wantErr:      false,
		},
		{
			name: "delete_timeout_single_config_single_item",
			args: &model.Timeout{
				Service: "DemoService",
				Group:   "",
				Version: "",
				Timeout: 6000,
			},
			existingRule: "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      timeout: 5000",
			want:         "",
			wantDelete:   true,
			wantErr:      false,
		},
		{
			name: "delete_timeout_single_config_multi_items",
			args: &model.Timeout{
				Service: "DemoService",
				Group:   "",
				Version: "",
				Timeout: 6000,
			},
			existingRule: "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      timeout: 5000\n      test: \"value\"",
			want:         "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      test: \"value\"",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeoutSvc := &TimeoutService{}

			ctrl := gomock.NewController(t)
			mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
			if tt.wantDelete {
				mockGovernanceConfig.EXPECT().DeleteConfig(gomock.Any()).Times(1)
			} else {
				mockGovernanceConfig.EXPECT().SetConfig(gomock.Any(), gomock.Any()).Do(func(key, rule string) {
					capturedRule = rule
				})
			}
			mockGovernanceConfig.EXPECT().GetConfig(gomock.Any()).Return(tt.existingRule, nil)
			config.Governance = mockGovernanceConfig

			if err := timeoutSvc.Delete(tt.args); err != nil && !tt.wantErr {
				t.Errorf("TimeoutService.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantDelete {
				actualJson, _ := yaml.YAMLToJSON([]byte(capturedRule))
				wantedJson, _ := yaml.YAMLToJSON([]byte(tt.want))
				if !jsonpatch.Equal(actualJson, wantedJson) {
					t.Errorf("TimeoutService.CreateOrUpdate() error \n, expected:\n %v \n, got:\n  %v \n", tt.want, capturedRule)
				}
			}
		})
	}
}

func TestSearchTimeout(t *testing.T) {
	tests := []struct {
		name         string
		args         *model.Timeout
		want         []*model.Timeout
		existingRule string
		wantErr      bool
	}{
		{
			name: "search_timeout_multi_configs",
			args: &model.Timeout{
				Service: "DemoService",
				Group:   "",
				Version: "",
				Timeout: 5000,
			},
			existingRule: "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      timeout: 5000\n      test: \"value\"\n  - side: provider\n    enabled: true\n    parameters:\n      accesslog: true",
			want: []*model.Timeout{{
				Service: "DemoService",
				Timeout: 5000,
			}},
			wantErr: false,
		},
		{
			name: "search_timeout_single_config_single_item",
			args: &model.Timeout{
				Service: "DemoService",
				Group:   "",
				Version: "",
				Timeout: 6000,
			},
			existingRule: "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      timeout: 5000",
			want: []*model.Timeout{{
				Service: "DemoService",
				Timeout: 5000,
			}},
			wantErr: false,
		},
		{
			name: "search_timeout_single_config_multi_items",
			args: &model.Timeout{
				Service: "DemoService",
				Group:   "",
				Version: "",
				Timeout: 6000,
			},
			existingRule: "configVersion: v3.0\nscope: service\nenabled: true\nkey: DemoService\nconfigs:\n  - side: consumer\n    enabled: true\n    parameters:\n      timeout: 5000\n      test: \"value\"",
			want: []*model.Timeout{{
				Service: "DemoService",
				Timeout: 5000,
			}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeoutSvc := &TimeoutService{}

			var capturedRule string
			ctrl := gomock.NewController(t)
			mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
			mockGovernanceConfig.EXPECT().GetConfig(gomock.Any()).DoAndReturn(func(key string) (string, error) {
				capturedRule = tt.existingRule
				return tt.existingRule, nil
			})
			fmt.Print(capturedRule)
			config.Governance = mockGovernanceConfig

			if timeouts, err := timeoutSvc.Search(tt.args); err != nil && !tt.wantErr {
				t.Errorf("TimeoutService.Search() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				if !reflect.DeepEqual(timeouts, tt.want) {
					t.Errorf("TimeoutService.Search() got = %v, want %v", timeouts, tt.want)
				}
			}
		})
	}
}
