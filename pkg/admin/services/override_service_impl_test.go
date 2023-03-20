// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//

//	http://www.apache.org/licenses/LICENSE-2.0

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

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/golang/mock/gomock"
)

func TestOverrideServiceImpl_SaveOverride(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().SetConfig(gomock.Any(), gomock.Any()).Return(nil)
	mockGovernanceConfig.EXPECT().GetConfig(gomock.Any()).Return("", nil)
	mockGovernanceConfig.EXPECT().Register(gomock.Any()).Return(nil)

	type args struct {
		dynamicConfig *model.DynamicConfig
	}
	tests := []struct {
		name    string
		s       OverrideService
		args    args
		wantErr bool
	}{
		{
			name: "TestOK",
			s: &OverrideServiceImpl{
				GovernanceConfig: mockGovernanceConfig,
			},
			args: args{
				dynamicConfig: &model.DynamicConfig{
					Base: model.Base{
						Application:    "",
						Service:        "testService",
						ServiceGroup:   "testGroup",
						ServiceVersion: "1.2.3",
					},
					Enabled:       true,
					ConfigVersion: "v2.7",
					Configs: []model.OverrideConfig{
						{
							Addresses: []string{"0.0.0.0"},
							Parameters: map[string]string{
								"timeout": "1000",
							},
							Side: "consumer",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.SaveOverride(tt.args.dynamicConfig); (err != nil) != tt.wantErr {
				t.Errorf("OverrideServiceImpl.SaveOverride() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOverrideServiceImpl_UpdateOverride(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().SetConfig(gomock.Any(), gomock.Any()).Return(nil)
	mockGovernanceConfig.EXPECT().GetConfig(getPath("testGroup/testService:testVersion")).Return("configVersion: v2.7\nconfigs:\n- addresses:\n  - 0.0.0.0\n  enabled: false\n  parameters:\n    timeout: 6000\n  side: consumer\nenabled: true\nkey: testService\nscope: service\n", nil)
	mockGovernanceConfig.EXPECT().Register(gomock.Any()).Return(nil)
	mockGovernanceConfig.EXPECT().UnRegister(gomock.Any()).Return(nil)

	type args struct {
		update *model.DynamicConfig
	}
	tests := []struct {
		name    string
		s       *OverrideServiceImpl
		args    args
		wantErr bool
	}{
		{
			name: "TestOK",
			s: &OverrideServiceImpl{
				GovernanceConfig: mockGovernanceConfig,
			},
			args: args{
				update: &model.DynamicConfig{
					Base: model.Base{
						Application:    "",
						Service:        "testService",
						ServiceGroup:   "testGroup",
						ServiceVersion: "testVersion",
					},
					Enabled:       true,
					ConfigVersion: "v2.7",
					Configs: []model.OverrideConfig{
						{
							Addresses: []string{"0.0.0.0"},
							Parameters: map[string]string{
								"timeout": "1000",
							},
							Side: "consumer",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.UpdateOverride(tt.args.update); (err != nil) != tt.wantErr {
				t.Errorf("OverrideServiceImpl.UpdateOverride() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOverrideServiceImpl_FindOverride(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().GetConfig(getPath("testGroup/testService:testVersion")).Return("configVersion: v2.7\nconfigs:\n- addresses:\n  - 0.0.0.0\n  enabled: false\n  parameters:\n    timeout: 6000\n  side: consumer\nenabled: true\nkey: testService\nscope: service\n", nil)

	type args struct {
		key string
	}
	tests := []struct {
		name    string
		s       *OverrideServiceImpl
		args    args
		want    *model.DynamicConfig
		wantErr bool
	}{
		{
			name: "TestOK",
			s: &OverrideServiceImpl{
				GovernanceConfig: mockGovernanceConfig,
			},
			args: args{
				key: "testGroup/testService:testVersion",
			},
			want: &model.DynamicConfig{
				Base: model.Base{
					ID:             "testGroup/testService:testVersion",
					Service:        "testService",
					ServiceGroup:   "testGroup",
					ServiceVersion: "testVersion",
				},
				ConfigVersion: "v2.7",
				Enabled:       true,
				Configs: []model.OverrideConfig{
					{
						Addresses: []string{"0.0.0.0"},
						Parameters: map[string]string{
							"timeout": "6000",
						},
						Enabled: false,
						Side:    "consumer",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.FindOverride(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("OverrideServiceImpl.FindOverride() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OverrideServiceImpl.FindOverride() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestOverrideServiceImpl_DeleteOverride(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().GetConfig(getPath("testGroup/testService:testVersion")).Return("configVersion: v2.7\nconfigs:\n- addresses:\n  - 0.0.0.0\n  enabled: false\n  parameters:\n    timeout: 6000\n  side: consumer\nenabled: true\nkey: testService\nscope: service\n", nil)
	mockGovernanceConfig.EXPECT().DeleteConfig(getPath("testGroup/testService:testVersion")).Return(nil)
	mockGovernanceConfig.EXPECT().UnRegister(gomock.Any()).Return(nil)

	type args struct {
		key string
	}
	tests := []struct {
		name    string
		s       *OverrideServiceImpl
		args    args
		wantErr bool
	}{
		{
			name: "TestOK",
			s: &OverrideServiceImpl{
				GovernanceConfig: mockGovernanceConfig,
			},
			args: args{
				key: "testGroup/testService:testVersion",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.DeleteOverride(tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("OverrideServiceImpl.DeleteOverride() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOverrideServiceImpl_EnableOverride(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().GetConfig(getPath("testGroup/testService:testVersion")).Return("configVersion: v2.7\nconfigs:\n- addresses:\n  - 0.0.0.0\n  enabled: false\n  parameters:\n    timeout: 6000\n  side: consumer\nenabled: true\nkey: testService\nscope: service\n", nil)
	mockGovernanceConfig.EXPECT().SetConfig(getPath("testGroup/testService:testVersion"), gomock.Any()).Return(nil)
	mockGovernanceConfig.EXPECT().Register(gomock.Any()).Return(nil)
	mockGovernanceConfig.EXPECT().UnRegister(gomock.Any()).Return(nil)

	type args struct {
		key string
	}
	tests := []struct {
		name    string
		s       *OverrideServiceImpl
		args    args
		wantErr bool
	}{
		{
			name: "TestOK",
			s: &OverrideServiceImpl{
				GovernanceConfig: mockGovernanceConfig,
			},
			args: args{
				key: "testGroup/testService:testVersion",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.EnableOverride(tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("OverrideServiceImpl.EnableOverride() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOverrideServiceImpl_DisableOverride(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().GetConfig(getPath("testGroup/testService:testVersion")).Return("configVersion: v2.7\nconfigs:\n- addresses:\n  - 0.0.0.0\n  enabled: false\n  parameters:\n    timeout: 6000\n  side: consumer\nenabled: true\nkey: testService\nscope: service\n", nil)
	mockGovernanceConfig.EXPECT().SetConfig(getPath("testGroup/testService:testVersion"), gomock.Any()).Return(nil)
	mockGovernanceConfig.EXPECT().Register(gomock.Any()).Return(nil)
	mockGovernanceConfig.EXPECT().UnRegister(gomock.Any()).Return(nil)

	type args struct {
		key string
	}
	tests := []struct {
		name    string
		s       *OverrideServiceImpl
		args    args
		wantErr bool
	}{
		{
			name: "TestOK",
			s: &OverrideServiceImpl{
				GovernanceConfig: mockGovernanceConfig,
			},
			args: args{
				key: "testGroup/testService:testVersion",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.DisableOverride(tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("OverrideServiceImpl.DisableOverride() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
