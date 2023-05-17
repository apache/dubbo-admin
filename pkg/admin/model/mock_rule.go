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

package model

import (
	"gorm.io/gorm"
)

type MockRule struct {
	ID          uint   `json:"id"`
	ServiceName string `json:"serviceName"`
	MethodName  string `json:"methodName"`
	Rule        string `json:"rule"`
	Enable      bool   `json:"enable"`
}

func (m *MockRule) ToMockRuleEntity() *MockRuleEntity {
	return &MockRuleEntity{
		Model: gorm.Model{
			ID: m.ID,
		},
		ServiceName: m.ServiceName,
		MethodName:  m.MethodName,
		Rule:        m.Rule,
		Enable:      m.Enable,
	}
}

type MockRuleEntity struct {
	gorm.Model
	ServiceName string `gorm:"type:varchar(255)"`
	MethodName  string `gorm:"type:varchar(255)"`
	Rule        string `gorm:"type:text"`
	Enable      bool
}

func (m *MockRuleEntity) ToMockRule() *MockRule {
	return &MockRule{
		ID:          uint(m.ID),
		ServiceName: m.ServiceName,
		MethodName:  m.MethodName,
		Rule:        m.Rule,
		Enable:      m.Enable,
	}
}

func (m *MockRuleEntity) TableName() string {
	return "mock_rule"
}

type ListMockRulesByPage struct {
	Total   int64       `json:"total"`
	Content []*MockRule `json:"content"`
}
