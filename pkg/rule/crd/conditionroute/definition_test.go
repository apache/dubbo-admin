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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCopy(t *testing.T) {
	policy := &Policy{
		Name: "test-policy",
		Spec: &PolicySpec{
			Priority: 0,
			Enabled:  true,
			Force:    true,
			Runtime:  true,
			Key:      "org.apache.dubbo.samples.CommentService",
			Scope:    "service",
			Conditions: []string{
				"method=getComment => region=Hangzhou",
			},
			ConfigVersion: "v3.0",
		},
	}

	toClient := policy.CopyToClient()

	assert.Equal(t, policy.Name, toClient.Name)

	assert.Equal(t, policy.Spec.Priority, toClient.Spec.Priority)
	assert.Equal(t, policy.Spec.Enabled, toClient.Spec.Enabled)
	assert.Equal(t, policy.Spec.Force, toClient.Spec.Force)
	assert.Equal(t, policy.Spec.Runtime, toClient.Spec.Runtime)
	assert.Equal(t, policy.Spec.Key, toClient.Spec.Key)
	assert.Equal(t, policy.Spec.Scope, toClient.Spec.Scope)
	assert.Equal(t, policy.Spec.Conditions[0], toClient.Spec.Conditions[0])
	assert.Equal(t, policy.Spec.ConfigVersion, toClient.Spec.ConfigVersion)
}
