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

package tagroute

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopy(t *testing.T) {
	policy := &Policy{
		Name: "test-policy",
		Spec: &PolicySpec{
			Priority: 0,
			Enabled:  true,
			Force:    true,
			Runtime:  true,
			Key:      "test-key",
			Tags: []*Tag{
				{
					Name:      "test-name",
					Addresses: []string{"test-address"},
					Match: []*ParamMatch{
						{
							Key: "test-key",
							Value: &StringMatch{
								Exact:    "test-exact1",
								Prefix:   "test-prefix1",
								Regex:    "test-regex1",
								Noempty:  "test-noempty1",
								Empty:    "test-empty1",
								Wildcard: "test-wildcard1",
							},
						},
					},
				},
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
	assert.Equal(t, policy.Spec.ConfigVersion, toClient.Spec.ConfigVersion)
	assert.Equal(t, policy.Spec.Tags[0].Name, toClient.Spec.Tags[0].Name)
	assert.Equal(t, policy.Spec.Tags[0].Addresses[0], toClient.Spec.Tags[0].Addresses[0])
	assert.Equal(t, policy.Spec.Tags[0].Match[0].Key, toClient.Spec.Tags[0].Match[0].Key)
	assert.Equal(t, policy.Spec.Tags[0].Match[0].Value.Exact, toClient.Spec.Tags[0].Match[0].Value.Exact)
	assert.Equal(t, policy.Spec.Tags[0].Match[0].Value.Prefix, toClient.Spec.Tags[0].Match[0].Value.Prefix)
	assert.Equal(t, policy.Spec.Tags[0].Match[0].Value.Regex, toClient.Spec.Tags[0].Match[0].Value.Regex)
	assert.Equal(t, policy.Spec.Tags[0].Match[0].Value.Noempty, toClient.Spec.Tags[0].Match[0].Value.Noempty)
	assert.Equal(t, policy.Spec.Tags[0].Match[0].Value.Empty, toClient.Spec.Tags[0].Match[0].Value.Empty)
	assert.Equal(t, policy.Spec.Tags[0].Match[0].Value.Wildcard, toClient.Spec.Tags[0].Match[0].Value.Wildcard)
}
