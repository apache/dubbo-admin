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

package mapper

import (
	"context"

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/model"
)

type MockRuleMapper interface {
	Create(mockRule *model.MockRuleEntity) error
	Update(mockRule *model.MockRuleEntity) error
	DeleteById(id int64) error
	FindByServiceNameAndMethodName(ctx context.Context, serviceName, methodName string) (*model.MockRuleEntity, error)
	FindByPage(filter string, offset, limit int) ([]*model.MockRuleEntity, int64, error)
}

type MockRuleMapperImpl struct{}

func (m *MockRuleMapperImpl) Create(mockRule *model.MockRuleEntity) error {
	return config.DataBase.Create(mockRule).Error
}

func (m *MockRuleMapperImpl) Update(mockRule *model.MockRuleEntity) error {
	return config.DataBase.Updates(mockRule).Error
}

func (m *MockRuleMapperImpl) DeleteById(id int64) error {
	return config.DataBase.Delete(&model.MockRuleEntity{}, id).Error
}

func (m *MockRuleMapperImpl) FindByServiceNameAndMethodName(ctx context.Context, serviceName, methodName string) (*model.MockRuleEntity, error) {
	var mockRule model.MockRuleEntity
	err := config.DataBase.WithContext(ctx).Where("service_name = ? and method_name = ?", serviceName, methodName).Limit(1).Find(&mockRule).Error
	return &mockRule, err
}

func (m *MockRuleMapperImpl) FindByPage(filter string, offset, limit int) ([]*model.MockRuleEntity, int64, error) {
	var mockRules []*model.MockRuleEntity
	var total int64
	err := config.DataBase.Where("service_name like ?", "%"+filter+"%").Offset(offset).Limit(limit).Find(&mockRules).Error
	config.DataBase.Model(&model.MockRuleEntity{}).Where("service_name like ?", "%"+filter+"%").Count(&total)
	return mockRules, total, err
}
