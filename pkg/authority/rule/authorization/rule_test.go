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

package authorization_test

import (
	"testing"

	"github.com/apache/dubbo-admin/pkg/authority/rule"
	"github.com/apache/dubbo-admin/pkg/authority/rule/authorization"
	"github.com/apache/dubbo-admin/pkg/authority/rule/connection"
)

func TestRule(t *testing.T) {
	t.Parallel()

	storage := connection.NewStorage()
	handler := authorization.NewHandler(storage)

	handler.Add("test", &authorization.Policy{})

	originRule := storage.LatestRules[authorization.RuleType]

	if originRule == nil {
		t.Error("expected origin rule to be added")
	}

	if originRule.Type() != authorization.RuleType {
		t.Error("expected origin rule type to be authorization/v1beta1")
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

	if toClient.Type() != authorization.RuleType {
		t.Error("expected toClient type to be authorization/v1beta1")
	}

	if toClient.Revision() != 1 {
		t.Error("expected toClient revision to be 1")
	}

	if toClient.Data() != `[{"spec":null}]` {
		t.Error("expected toClient data to be [{\"spec\":null}], got " + toClient.Data())
	}

	handler.Add("test2", &authorization.Policy{
		Name: "test2",
		Spec: &authorization.PolicySpec{
			Action: "ALLOW",
		},
	})

	originRule = storage.LatestRules[authorization.RuleType]

	if originRule == nil {
		t.Error("expected origin rule to be added")
	}

	if originRule.Type() != authorization.RuleType {
		t.Error("expected origin rule type to be authorization/v1beta1")
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

	if toClient.Type() != authorization.RuleType {
		t.Error("expected toClient type to be authorization/v1beta1")
	}

	if toClient.Revision() != 2 {
		t.Error("expected toClient revision to be 2")
	}

	if toClient.Data() != `[{"spec":null},{"name":"test2","spec":{"action":"ALLOW"}}]` {
		t.Error("expected toClient data to be [{\"spec\":null},{\"name\":\"test2\",\"spec\":{\"action\":\"ALLOW\"}}], got " + toClient.Data())
	}
}
