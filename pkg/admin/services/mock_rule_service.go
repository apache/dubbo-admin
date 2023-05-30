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

	"github.com/apache/dubbo-admin/pkg/admin/model"
)

type MockRuleService interface {
	// create or update mock rule. if the request contains id, then will be an update operation.
	CreateOrUpdateMockRule(mockRule *model.MockRule) error

	// delete the mock rule data by mock rule id.
	DeleteMockRuleById(id int64) error

	// list the mock rules by filter and return data by page.
	ListMockRulesByPage(filter string, offset, limit int) ([]*model.MockRule, int64, error)

	// TODO getMockData method, which depends on the implementation corresponding to mock of dubbo-go.
	GetMockData(ctx context.Context, serviceName, methodName string) (rule string, enable bool, err error)
}
