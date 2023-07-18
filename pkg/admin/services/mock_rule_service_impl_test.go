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

package services

import (
	"context"
	"reflect"
	"testing"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/apache/dubbo-admin/pkg/admin/mapper"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/golang/mock/gomock"
)

func TestMockRuleServiceImpl_CreateOrUpdateMockRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockMockRuleMapper := mapper.NewMockMockRuleMapper(ctrl)

	mockUnExistData := &model.MockRule{}
	createData := &model.MockRule{ID: 1, ServiceName: "testService1", MethodName: "testMethod1", Rule: "exampleRule", Enable: true}
	mockExistData := &model.MockRule{ID: 1, ServiceName: "testService2", MethodName: "testMethod2", Rule: "exampleRule", Enable: true}
	updateData := &model.MockRule{ID: 1, ServiceName: "testService2", MethodName: "testMethod2", Rule: "exampleRuleAfterUpdate", Enable: true}

	mockMockRuleMapper.EXPECT().FindByServiceNameAndMethodName(context.Background(), createData.ServiceName, createData.MethodName).Return(mockUnExistData.ToMockRuleEntity(), nil)
	mockMockRuleMapper.EXPECT().Create(createData.ToMockRuleEntity()).Return(nil)
	mockMockRuleMapper.EXPECT().FindByServiceNameAndMethodName(context.Background(), mockExistData.ServiceName, mockExistData.MethodName).Return(mockExistData.ToMockRuleEntity(), nil)
	mockMockRuleMapper.EXPECT().Update(updateData.ToMockRuleEntity()).Return(nil)

	type args struct {
		mockRule *model.MockRule
	}
	tests := []struct {
		name    string
		m       *MockRuleServiceImpl
		args    args
		wantErr bool
	}{
		{
			name: "test create mock rule",
			m: &MockRuleServiceImpl{
				MockRuleMapper: mockMockRuleMapper,
				Logger:         logger.Logger(),
			},
			args: args{
				mockRule: createData,
			},
			wantErr: false,
		},
		{
			name: "test update mock rule",
			m: &MockRuleServiceImpl{
				MockRuleMapper: mockMockRuleMapper,
				Logger:         logger.Logger(),
			},
			args: args{
				mockRule: updateData,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.m.CreateOrUpdateMockRule(tt.args.mockRule); (err != nil) != tt.wantErr {
				t.Errorf("MockRuleServiceImpl.CreateOrUpdateMockRule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMockRuleServiceImpl_DeleteMockRuleById(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockMockRuleMapper := mapper.NewMockMockRuleMapper(ctrl)

	mockMockRuleMapper.EXPECT().DeleteById(int64(1)).Return(nil)

	type args struct {
		id int64
	}
	tests := []struct {
		name    string
		m       *MockRuleServiceImpl
		args    args
		wantErr bool
	}{
		{
			name: "test delete mock rule",
			m: &MockRuleServiceImpl{
				MockRuleMapper: mockMockRuleMapper,
				Logger:         logger.Logger(),
			},
			args: args{
				id: 1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.m.DeleteMockRuleById(tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("MockRuleServiceImpl.DeleteMockRuleById() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMockRuleServiceImpl_ListMockRulesByPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockMockRuleMapper := mapper.NewMockMockRuleMapper(ctrl)

	findResult := &model.MockRule{ID: 1, ServiceName: "testService2", MethodName: "testMethod2", Rule: "exampleRule", Enable: true}
	mockMockRuleMapper.EXPECT().FindByPage("", 0, -1).Return([]*model.MockRuleEntity{findResult.ToMockRuleEntity()}, int64(1), nil)

	type args struct {
		filter string
		offset int
		limit  int
	}
	tests := []struct {
		name    string
		m       *MockRuleServiceImpl
		args    args
		want    []*model.MockRule
		want1   int64
		wantErr bool
	}{
		{
			name: "test list mock rule",
			m: &MockRuleServiceImpl{
				MockRuleMapper: mockMockRuleMapper,
				Logger:         logger.Logger(),
			},
			args: args{
				filter: "",
				offset: 0,
				limit:  -1,
			},
			want:    []*model.MockRule{findResult},
			want1:   1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := tt.m.ListMockRulesByPage(tt.args.filter, tt.args.offset, tt.args.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("MockRuleServiceImpl.ListMockRulesByPage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MockRuleServiceImpl.ListMockRulesByPage() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("MockRuleServiceImpl.ListMockRulesByPage() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
