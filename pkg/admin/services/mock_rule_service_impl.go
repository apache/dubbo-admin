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
	"errors"

	"github.com/apache/dubbo-admin/pkg/admin/mapper"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"go.uber.org/zap"
)

type MockRuleServiceImpl struct {
	MockRuleMapper mapper.MockRuleMapper
	Logger         *zap.Logger
}

func (m *MockRuleServiceImpl) CreateOrUpdateMockRule(mockRule *model.MockRule) error {
	if mockRule.ServiceName == "" || mockRule.MethodName == "" || mockRule.Rule == "" {
		return nil
	}

	existRule, err := m.MockRuleMapper.FindByServiceNameAndMethodName(context.TODO(), mockRule.ServiceName, mockRule.MethodName)
	if err != nil {
		m.Logger.Error(err.Error())
		return err
	}

	isExist := existRule.ID != 0
	// check if we can save or update the rule, we need keep the serviceName + methodName is unique.
	if isExist {
		if mockRule.ID != existRule.ID {
			err := errors.New("service name and method name must be unique")
			m.Logger.Error(err.Error())
			return err
		}
		if err := m.MockRuleMapper.Update(mockRule.ToMockRuleEntity()); err != nil {
			m.Logger.Error(err.Error())
			return err
		}
	} else {
		if err := m.MockRuleMapper.Create(mockRule.ToMockRuleEntity()); err != nil {
			m.Logger.Error(err.Error())
			return err
		}
	}
	return nil
}

func (m *MockRuleServiceImpl) DeleteMockRuleById(id int64) error {
	if err := m.MockRuleMapper.DeleteById(id); err != nil {
		m.Logger.Error(err.Error())
		return err
	}
	return nil
}

func (m *MockRuleServiceImpl) ListMockRulesByPage(filter string, offset, limit int) ([]*model.MockRule, int64, error) {
	result, total, err := m.MockRuleMapper.FindByPage(filter, offset, limit)
	if err != nil {
		m.Logger.Error(err.Error())
		return nil, 0, err
	}

	morkRules := make([]*model.MockRule, 0)
	for _, mockRuleEntity := range result {
		morkRules = append(morkRules, mockRuleEntity.ToMockRule())
	}
	return morkRules, total, nil
}

func (m *MockRuleServiceImpl) GetMockData(ctx context.Context, serviceName, methodName string) (rule string, enable bool, err error) {
	mockRule, err := m.MockRuleMapper.FindByServiceNameAndMethodName(ctx, serviceName, methodName)
	if err != nil {
		m.Logger.Error(err.Error())
		return "", false, err
	}
	return mockRule.Rule, mockRule.Enable, nil
}
