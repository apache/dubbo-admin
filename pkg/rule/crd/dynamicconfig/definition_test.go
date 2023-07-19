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

package dynamicconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopy(t *testing.T) {
	policy := &Policy{
		Name: "test-policy",
		Spec: &PolicySpec{
			Key:           "org.apache.dubbo.samples.UserService",
			Scope:         "service",
			ConfigVersion: "v3.0",
			Enabled:       true,
			Configs: []*OverrideConfig{
				{
					Side: "consumer",
					Addresses: []string{
						"test-address",
					},
					ProviderAddresses: []string{
						"test-providerAddresses",
					},
					Parameters: map[string]string{
						"retries": "4",
					},
					Applications: []string{
						"test-applications",
					},
					Services: []string{
						"test-services",
					},
					Type:    "test-type",
					Enabled: true,
					Match: &ConditionMatch{
						Address: &AddressMatch{
							Wildcard: "test-wildcard",
							Cird:     "test-cird",
							Exact:    "test-exact",
						},
						Service: &ListStringMatch{
							Oneof: []*StringMatch{
								{
									Exact:    "test-exact1",
									Prefix:   "test-prefix1",
									Regex:    "test-regex1",
									Noempty:  "test-noempty1",
									Empty:    "test-empty1",
									Wildcard: "test-wildcard1",
								},
							},
						},
						Application: &ListStringMatch{
							Oneof: []*StringMatch{
								{
									Exact:    "test-exact2",
									Prefix:   "test-prefix2",
									Regex:    "test-regex2",
									Noempty:  "test-noempty2",
									Empty:    "test-empty2",
									Wildcard: "test-wildcard2",
								},
							},
						},
						Param: []*ParamMatch{
							{
								Key: "test-key",
								Value: &StringMatch{
									Exact:    "test-exact3",
									Prefix:   "test-prefix3",
									Regex:    "test-regex3",
									Noempty:  "test-noempty3",
									Empty:    "test-empty3",
									Wildcard: "test-wildcard3",
								},
							},
						},
					},
				},
			},
		},
	}

	toClient := policy.CopyToClient()

	assert.Equal(t, policy.Name, toClient.Name)
	assert.Equal(t, policy.Spec.Key, toClient.Spec.Key)
	assert.Equal(t, policy.Spec.Scope, toClient.Spec.Scope)
	assert.Equal(t, policy.Spec.ConfigVersion, toClient.Spec.ConfigVersion)
	assert.Equal(t, policy.Spec.Enabled, toClient.Spec.Enabled)
	assert.Equal(t, policy.Spec.Configs[0].Side, toClient.Spec.Configs[0].Side)
	assert.Equal(t, policy.Spec.Configs[0].Addresses[0], toClient.Spec.Configs[0].Addresses[0])
	assert.Equal(t, policy.Spec.Configs[0].ProviderAddresses[0], toClient.Spec.Configs[0].ProviderAddresses[0])
	assert.Equal(t, policy.Spec.Configs[0].Parameters["retries"], toClient.Spec.Configs[0].Parameters["retries"])
	assert.Equal(t, policy.Spec.Configs[0].Applications[0], toClient.Spec.Configs[0].Applications[0])
	assert.Equal(t, policy.Spec.Configs[0].Services[0], toClient.Spec.Configs[0].Services[0])
	assert.Equal(t, policy.Spec.Configs[0].Type, toClient.Spec.Configs[0].Type)
	assert.Equal(t, policy.Spec.Configs[0].Enabled, toClient.Spec.Configs[0].Enabled)
	assert.Equal(t, policy.Spec.Configs[0].Match.Address.Wildcard, toClient.Spec.Configs[0].Match.Address.Wildcard)
	assert.Equal(t, policy.Spec.Configs[0].Match.Address.Cird, toClient.Spec.Configs[0].Match.Address.Cird)
	assert.Equal(t, policy.Spec.Configs[0].Match.Address.Exact, toClient.Spec.Configs[0].Match.Address.Exact)
	assert.Equal(t, policy.Spec.Configs[0].Match.Service.Oneof[0].Exact, toClient.Spec.Configs[0].Match.Service.Oneof[0].Exact)
	assert.Equal(t, policy.Spec.Configs[0].Match.Service.Oneof[0].Prefix, toClient.Spec.Configs[0].Match.Service.Oneof[0].Prefix)
	assert.Equal(t, policy.Spec.Configs[0].Match.Service.Oneof[0].Regex, toClient.Spec.Configs[0].Match.Service.Oneof[0].Regex)
	assert.Equal(t, policy.Spec.Configs[0].Match.Service.Oneof[0].Empty, toClient.Spec.Configs[0].Match.Service.Oneof[0].Empty)
	assert.Equal(t, policy.Spec.Configs[0].Match.Service.Oneof[0].Noempty, toClient.Spec.Configs[0].Match.Service.Oneof[0].Noempty)
	assert.Equal(t, policy.Spec.Configs[0].Match.Service.Oneof[0].Wildcard, toClient.Spec.Configs[0].Match.Service.Oneof[0].Wildcard)

	assert.Equal(t, policy.Spec.Configs[0].Match.Application.Oneof[0].Exact, toClient.Spec.Configs[0].Match.Application.Oneof[0].Exact)
	assert.Equal(t, policy.Spec.Configs[0].Match.Application.Oneof[0].Prefix, toClient.Spec.Configs[0].Match.Application.Oneof[0].Prefix)
	assert.Equal(t, policy.Spec.Configs[0].Match.Application.Oneof[0].Regex, toClient.Spec.Configs[0].Match.Application.Oneof[0].Regex)
	assert.Equal(t, policy.Spec.Configs[0].Match.Application.Oneof[0].Empty, toClient.Spec.Configs[0].Match.Application.Oneof[0].Empty)
	assert.Equal(t, policy.Spec.Configs[0].Match.Application.Oneof[0].Noempty, toClient.Spec.Configs[0].Match.Application.Oneof[0].Noempty)
	assert.Equal(t, policy.Spec.Configs[0].Match.Application.Oneof[0].Wildcard, toClient.Spec.Configs[0].Match.Application.Oneof[0].Wildcard)

	assert.Equal(t, policy.Spec.Configs[0].Match.Param[0].Key, toClient.Spec.Configs[0].Match.Param[0].Key)
	assert.Equal(t, policy.Spec.Configs[0].Match.Param[0].Value.Exact, toClient.Spec.Configs[0].Match.Param[0].Value.Exact)
	assert.Equal(t, policy.Spec.Configs[0].Match.Param[0].Value.Prefix, toClient.Spec.Configs[0].Match.Param[0].Value.Prefix)
	assert.Equal(t, policy.Spec.Configs[0].Match.Param[0].Value.Regex, toClient.Spec.Configs[0].Match.Param[0].Value.Regex)
	assert.Equal(t, policy.Spec.Configs[0].Match.Param[0].Value.Noempty, toClient.Spec.Configs[0].Match.Param[0].Value.Noempty)
	assert.Equal(t, policy.Spec.Configs[0].Match.Param[0].Value.Empty, toClient.Spec.Configs[0].Match.Param[0].Value.Empty)
	assert.Equal(t, policy.Spec.Configs[0].Match.Param[0].Value.Wildcard, toClient.Spec.Configs[0].Match.Param[0].Value.Wildcard)
}
