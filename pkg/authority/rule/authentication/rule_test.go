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

package authentication_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apache/dubbo-admin/pkg/authority/rule"
	"github.com/apache/dubbo-admin/pkg/authority/rule/authentication"
	"github.com/apache/dubbo-admin/pkg/authority/rule/connection"
)

func TestRule(t *testing.T) {
	t.Parallel()

	storage := connection.NewStorage()
	handler := authentication.NewHandler(storage)

	handler.Add("test", &authentication.Policy{})

	originRule := storage.LatestRules[authentication.RuleType]

	if originRule == nil {
		t.Error("expected origin rule to be added")
	}

	if originRule.Type() != authentication.RuleType {
		t.Error("expected origin rule type to be authentication/v1beta1")
	}

	if originRule.Revision() != 1 {
		t.Error("expected origin rule revision to be 1")
	}

	toClient, err := originRule.Exact(&rule.Endpoint{
		ID:  "test",
		Ips: []string{"127.0.0.1"},
	})
	if err != nil {
		t.Error(err)
	}

	if toClient.Type() != authentication.RuleType {
		t.Error("expected toClient type to be authentication/v1beta1")
	}

	if toClient.Revision() != 1 {
		t.Error("expected toClient revision to be 1")
	}

	if toClient.Data() != `[{"spec":null}]` {
		t.Error("expected toClient data to be [{\"spec\":null}], got " + toClient.Data())
	}

	policy := &authentication.Policy{
		Name: "test2",
		Spec: &authentication.PolicySpec{
			Action: "ALLOW",
		},
	}

	handler.Add("test2", policy)

	originRule = storage.LatestRules[authentication.RuleType]

	if originRule == nil {
		t.Error("expected origin rule to be added")
	}

	if originRule.Type() != authentication.RuleType {
		t.Error("expected origin rule type to be authentication/v1beta1")
	}

	if originRule.Revision() != 2 {
		t.Error("expected origin rule revision to be 2")
	}

	toClient, err = originRule.Exact(&rule.Endpoint{
		ID:  "test",
		Ips: []string{"127.0.0.1"},
	})

	if err != nil {
		t.Error(err)
	}

	if toClient.Type() != authentication.RuleType {
		t.Error("expected toClient type to be authentication/v1beta1")
	}

	if toClient.Revision() != 2 {
		t.Error("expected toClient revision to be 2")
	}

	target := []*authentication.Policy{}

	err = json.Unmarshal([]byte(toClient.Data()), &target)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(target))

	assert.Contains(t, target, &authentication.Policy{})
	assert.Contains(t, target, policy)
}
