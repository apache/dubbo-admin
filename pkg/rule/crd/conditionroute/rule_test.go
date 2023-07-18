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

package conditionroute

import (
	"encoding/json"
	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/rule/storage"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRule(t *testing.T) {
	t.Parallel()

	storages := storage.NewStorage()
	handler := NewHandler(storages)

	handler.Add("test", &Policy{})

	originRule := storages.LatestRules[storage.ConditionRoute]

	if originRule == nil {
		t.Error("expected origin rule to be added")
	}

	if originRule.Type() != storage.ConditionRoute {
		t.Error("expected origin rule type to be authorization/v1beta1")
	}

	if originRule.Revision() != 1 {
		t.Error("expected origin rule revision to be 1")
	}

	toClient, err := originRule.Exact(&endpoint.Endpoint{
		ID:  "test",
		Ips: []string{"127.0.0.1"},
	})
	if err != nil {
		t.Error(err)
	}

	if toClient.Type() != storage.ConditionRoute {
		t.Error("expected toClient type to be authorization/v1beta1")
	}

	if toClient.Revision() != 1 {
		t.Error("expected toClient revision to be 1")
	}

	if toClient.Data() != `[]` {
		t.Error("expected toClient data to be [], got " + toClient.Data())
	}

	policy := &Policy{
		Name: "test2",
		Spec: &PolicySpec{
			Key: "test-key",
		},
	}

	handler.Add("test2", policy)

	originRule = storages.LatestRules[storage.ConditionRoute]

	if originRule == nil {
		t.Error("expected origin rule to be added")
	}

	if originRule.Type() != storage.ConditionRoute {
		t.Error("expected origin rule type to be authorization/v1beta1")
	}

	if originRule.Revision() != 2 {
		t.Error("expected origin rule revision to be 2")
	}

	toClient, err = originRule.Exact(&endpoint.Endpoint{
		ID:  "test",
		Ips: []string{"127.0.0.1"},
	})

	if err != nil {
		t.Error(err)
	}

	if toClient.Type() != storage.ConditionRoute {
		t.Error("expected toClient type to be authorization/v1beta1")
	}

	if toClient.Revision() != 2 {
		t.Error("expected toClient revision to be 2")
	}

	target := []*Policy{}

	err = json.Unmarshal([]byte(toClient.Data()), &target)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(target))

	assert.Contains(t, target, policy)
}
