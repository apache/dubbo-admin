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

	"github.com/apache/dubbo-admin/pkg/admin/config/mock_config"

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/util"
	"github.com/golang/mock/gomock"
)

func TestRouteServiceImpl_CreateTagRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().SetConfig(gomock.Any(), gomock.Any()).Return(nil)
	config.Governance = mockGovernanceConfig

	type args struct {
		tagRoute *model.TagRouteDto
	}
	tests := []struct {
		name    string
		s       RouteService
		args    args
		wantErr bool
	}{
		{
			name: "test create tag route",
			s:    &RouteServiceImpl{},
			args: args{
				tagRoute: &model.TagRouteDto{
					Base: model.Base{
						Application:    "",
						Service:        "testService",
						ServiceVersion: "testVersion",
						ServiceGroup:   "testGroup",
					},
					Enabled:       true,
					ConfigVersion: "v3.0",
					Force:         true,
					Tags: []model.Tag{
						{
							Name: "gray",
							Match: []model.ParamMatch{
								{
									Key: "env",
									Value: model.StringMatch{
										Exact: "gray",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.CreateTagRoute(*tt.args.tagRoute); (err != nil) != tt.wantErr {
				t.Errorf("RouteServiceImpl.CreateTagRoute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRouteServiceImpl_UpdateTagRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().SetConfig(gomock.Any(), gomock.Any()).Return(nil)
	mockGovernanceConfig.EXPECT().GetConfig(GetRoutePath(util.BuildServiceKey("", "testService", "testVersion", "testGroup"), constant.TagRoute)).Return(`{"enabled":true,"force":true,"key":"testService:testVersion:testGroup","tags":[{"name":"gray","match":[{"key":"env","value":{"exact":"gray"}}]}]}`, nil)
	config.Governance = mockGovernanceConfig

	type args struct {
		tagRoute *model.TagRouteDto
	}
	tests := []struct {
		name    string
		s       RouteService
		args    args
		wantErr bool
	}{
		{
			name: "test update tag route",
			s:    &RouteServiceImpl{},
			args: args{
				tagRoute: &model.TagRouteDto{
					Base: model.Base{
						Application:    "",
						Service:        "testService",
						ServiceVersion: "testVersion",
						ServiceGroup:   "testGroup",
					},
					Enabled:       true,
					ConfigVersion: "v3.0",
					Force:         false,
					Tags: []model.Tag{
						{
							Name: "gray",
							Match: []model.ParamMatch{
								{
									Key: "env",
									Value: model.StringMatch{
										Exact: "gray",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.UpdateTagRoute(*tt.args.tagRoute); (err != nil) != tt.wantErr {
				t.Errorf("RouteServiceImpl.UpdateTagRoute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRouteServiceImpl_DeleteTagRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().DeleteConfig(GetRoutePath(util.BuildServiceKey("", "testService", "testVersion", "testGroup"), constant.TagRoute)).Return(nil)
	config.Governance = mockGovernanceConfig

	type args struct {
		key string
	}
	tests := []struct {
		name    string
		s       RouteService
		args    args
		wantErr bool
	}{
		{
			name: "test delete tag route",
			s:    &RouteServiceImpl{},
			args: args{
				key: "testService:testVersion:testGroup",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.DeleteTagRoute(tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("RouteServiceImpl.DeleteTagRoute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRouteServiceImpl_FindTagRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().GetConfig(GetRoutePath(util.BuildServiceKey("", "testService", "testVersion", "testGroup"), constant.TagRoute)).Return(`{"enabled":true,"force":true,"key":"testService:testVersion:testGroup","tags":[{"name":"gray","match":[{"key":"env","value":{"exact":"gray"}}]}]}`, nil)
	config.Governance = mockGovernanceConfig

	type args struct {
		key string
	}
	tests := []struct {
		name    string
		s       RouteService
		args    args
		want    model.TagRouteDto
		wantErr bool
	}{
		{
			name: "test find tag route",
			s:    &RouteServiceImpl{},
			args: args{
				key: "testService:testVersion:testGroup",
			},
			want: model.TagRouteDto{
				Base: model.Base{
					Application: "testService:testVersion:testGroup",
				},
				Enabled: true,
				Force:   true,
				Tags: []model.Tag{
					{
						Name: "gray",
						Match: []model.ParamMatch{
							{
								Key: "env",
								Value: model.StringMatch{
									Exact: "gray",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.FindTagRoute(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("RouteServiceImpl.FindTagRoute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RouteServiceImpl.FindTagRoute() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRouteServiceImpl_EnableTagRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().SetConfig(gomock.Any(), gomock.Any()).Return(nil)
	mockGovernanceConfig.EXPECT().GetConfig(GetRoutePath(util.BuildServiceKey("", "testService", "testVersion", "testGroup"), constant.TagRoute)).Return(`{"enabled":true,"force":true,"key":"testService:testVersion:testGroup","tags":[{"name":"gray","match":[{"key":"env","value":{"exact":"gray"}}]}]}`, nil)
	config.Governance = mockGovernanceConfig

	type args struct {
		key string
	}
	tests := []struct {
		name    string
		s       RouteService
		args    args
		wantErr bool
	}{
		{
			name: "test enable tag route",
			s:    &RouteServiceImpl{},
			args: args{
				key: "testService:testVersion:testGroup",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.EnableTagRoute(tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("RouteServiceImpl.EnableTagRoute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRouteServiceImpl_DisableTagRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().SetConfig(gomock.Any(), gomock.Any()).Return(nil)
	mockGovernanceConfig.EXPECT().GetConfig(GetRoutePath(util.BuildServiceKey("", "testService", "testVersion", "testGroup"), constant.TagRoute)).Return(`{"enabled":false,"force":true,"key":"testService:testVersion:testGroup","tags":[{"name":"gray","match":[{"key":"env","value":{"exact":"gray"}}]}]}`, nil)
	config.Governance = mockGovernanceConfig

	type args struct {
		key string
	}
	tests := []struct {
		name    string
		s       RouteService
		args    args
		wantErr bool
	}{
		{
			name: "test disable tag route",
			s:    &RouteServiceImpl{},
			args: args{
				key: "testService:testVersion:testGroup",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.DisableTagRoute(tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("RouteServiceImpl.DisableTagRoute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRouteServiceImpl_CreateConditionRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().SetConfig(gomock.Any(), gomock.Any()).Return(nil)
	mockGovernanceConfig.EXPECT().GetConfig(GetRoutePath(util.BuildServiceKey("", "testService", "testVersion", "testGroup"), constant.ConditionRoute)).Return("", nil)
	config.Governance = mockGovernanceConfig

	type args struct {
		route *model.ConditionRouteDto
	}
	tests := []struct {
		name    string
		s       RouteService
		args    args
		wantErr bool
	}{
		{
			name: "test create condition route",
			s:    &RouteServiceImpl{},
			args: args{
				route: &model.ConditionRouteDto{
					Base: model.Base{
						Application:    "",
						Service:        "testService",
						ServiceGroup:   "testGroup",
						ServiceVersion: "testVersion",
					},
					Enabled:    true,
					Force:      true,
					Runtime:    true,
					Conditions: []string{"method=getComment => region=Hangzhou"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.CreateConditionRoute(*tt.args.route); (err != nil) != tt.wantErr {
				t.Errorf("RouteServiceImpl.CreateConditionRoute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRouteServiceImpl_UpdateConditionRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().SetConfig(gomock.Any(), gomock.Any()).Return(nil)
	mockGovernanceConfig.EXPECT().GetConfig(GetRoutePath(util.BuildServiceKey("", "testService", "testVersion", "testGroup"), constant.ConditionRoute)).Return(`{"enabled":true,"force":false,"runtime":true,"key":"testService:testVersion:testGroup","conditions":["method=getComment => region=Hangzhou"]}`, nil)
	config.Governance = mockGovernanceConfig

	type args struct {
		route model.ConditionRouteDto
	}
	tests := []struct {
		name    string
		s       RouteService
		args    args
		wantErr bool
	}{
		{
			name: "test update condition route",
			s:    &RouteServiceImpl{},
			args: args{
				route: model.ConditionRouteDto{
					Base: model.Base{
						Application:    "",
						Service:        "testService",
						ServiceGroup:   "testGroup",
						ServiceVersion: "testVersion",
					},
					Enabled:    true,
					Force:      true,
					Runtime:    true,
					Conditions: []string{"method=getComment => region=Hangzhou"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.UpdateConditionRoute(tt.args.route); (err != nil) != tt.wantErr {
				t.Errorf("RouteServiceImpl.UpdateConditionRoute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRouteServiceImpl_DeleteConditionRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().DeleteConfig(gomock.Any()).Return(nil)
	config.Governance = mockGovernanceConfig

	type args struct {
		key string
	}
	tests := []struct {
		name    string
		s       RouteService
		args    args
		wantErr bool
	}{
		{
			name: "test delete condition route",
			s:    &RouteServiceImpl{},
			args: args{
				key: "testService:testVersion:testGroup",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.DeleteConditionRoute(tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("RouteServiceImpl.DeleteConditionRoute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRouteServiceImpl_FindConditionRouteById(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().GetConfig(GetRoutePath(util.BuildServiceKey("", "testService", "testVersion", "testGroup"), constant.ConditionRoute)).Return(`{"enabled":true,"force":true,"runtime":true,"key":"testService:testVersion:testGroup","conditions":["method=getComment => region=Hangzhou"]}`, nil)
	config.Governance = mockGovernanceConfig

	type args struct {
		key string
	}
	tests := []struct {
		name    string
		s       RouteService
		args    args
		want    model.ConditionRouteDto
		wantErr bool
	}{
		{
			name: "test find condition route by id",
			s:    &RouteServiceImpl{},
			args: args{
				key: "testService:testVersion:testGroup",
			},
			want: model.ConditionRouteDto{
				Base: model.Base{
					ID:             "testService:testVersion:testGroup",
					Service:        "testService:testVersion:testGroup",
					ServiceGroup:   "testGroup",
					ServiceVersion: "testVersion",
				},
				Enabled:    true,
				Force:      true,
				Runtime:    true,
				Conditions: []string{"method=getComment => region=Hangzhou"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.FindConditionRouteById(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("RouteServiceImpl.FindConditionRouteById() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RouteServiceImpl.FindConditionRouteById() = %v\n, want %v", got, tt.want)
			}
		})
	}
}

func TestRouteServiceImpl_EnableConditionRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().SetConfig(gomock.Any(), gomock.Any()).Return(nil)
	mockGovernanceConfig.EXPECT().GetConfig(GetRoutePath(util.BuildServiceKey("", "testService", "testVersion", "testGroup"), constant.ConditionRoute)).Return(`{"enabled":false,"force":true,"runtime":true,"key":"testService:testVersion:testGroup","conditions":["method=getComment => region=Hangzhou"]}`, nil)
	config.Governance = mockGovernanceConfig

	type args struct {
		key string
	}
	tests := []struct {
		name    string
		s       RouteService
		args    args
		wantErr bool
	}{
		{
			name: "test enable condition route",
			s:    &RouteServiceImpl{},
			args: args{
				key: "testService:testVersion:testGroup",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.EnableConditionRoute(tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("RouteServiceImpl.EnableConditionRoute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRouteServiceImpl_DisableConditionRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGovernanceConfig := mock_config.NewMockGovernanceConfig(ctrl)
	mockGovernanceConfig.EXPECT().SetConfig(gomock.Any(), gomock.Any()).Return(nil)
	mockGovernanceConfig.EXPECT().GetConfig(GetRoutePath(util.BuildServiceKey("", "testService", "testVersion", "testGroup"), constant.ConditionRoute)).Return(`{"enabled":true,"force":true,"runtime":true,"key":"testService:testVersion:testGroup","conditions":["method=getComment => region=Hangzhou"]}`, nil)
	config.Governance = mockGovernanceConfig

	type args struct {
		key string
	}
	tests := []struct {
		name    string
		s       RouteService
		args    args
		wantErr bool
	}{
		{
			name: "test disable condition route",
			s:    &RouteServiceImpl{},
			args: args{
				key: "testService:testVersion:testGroup",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.DisableConditionRoute(tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("RouteServiceImpl.DisableConditionRoute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRouteServiceImpl_getRoutePath(t *testing.T) {
	config.Governance = nil
	type args struct {
		key       string
		routeType string
	}
	tests := []struct {
		name string
		s    *RouteServiceImpl
		args args
		want string
	}{
		{
			name: "test get route path",
			s:    &RouteServiceImpl{},
			args: args{
				key:       "testService:testVersion:testGroup",
				routeType: constant.TagRoute,
			},
			want: "testService:testVersion:testGroup" + constant.TagRuleSuffix,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRoutePath(tt.args.key, tt.args.routeType); got != tt.want {
				t.Errorf("RouteServiceImpl.GetRoutePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
