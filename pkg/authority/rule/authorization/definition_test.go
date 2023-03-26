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

package authorization

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopy(t *testing.T) {
	policy := &Policy{
		Name: "test-policy",
		Spec: &PolicySpec{
			Action: "test-action",
			Rules: []*PolicyRule{
				{
					From: &Source{
						Namespaces:    []string{"test-namespace"},
						NotNamespaces: []string{"test-not-namespace"},
						IpBlocks:      []string{"test-ip-block"},
						NotIpBlocks:   []string{"test-not-ip-block"},
						Principals:    []string{"test-principal"},
						NotPrincipals: []string{"test-not-principal"},
						Extends: []*Extend{
							{
								Key:   "test-key",
								Value: "test-value",
							},
						},
						NotExtends: []*Extend{
							{
								Key:   "test-not-key",
								Value: "test-not-value",
							},
						},
					},
					To: &Target{
						Namespaces:    []string{"test-namespace"},
						NotNamespaces: []string{"test-not-namespace"},
						IpBlocks:      []string{"test-ip-block"},
						NotIpBlocks:   []string{"test-not-ip-block"},
						Principals:    []string{"test-principal"},
						NotPrincipals: []string{"test-not-principal"},
						Extends: []*Extend{
							{
								Key:   "test-key",
								Value: "test-value",
							},
						},
						NotExtends: []*Extend{
							{
								Key:   "test-not-key",
								Value: "test-not-value",
							},
						},
					},
					When: &Condition{
						Key: "test-key",
						Values: []*Match{
							{
								Type:  "test-type",
								Value: "test-value",
							},
						},
						NotValues: []*Match{
							{
								Type:  "test-not-type",
								Value: "test-not-value",
							},
						},
					},
				},
			},
			Samples:   0.5,
			Order:     0.5,
			MatchType: "test-match-type",
		},
	}

	toClient := policy.CopyToClient()

	assert.Equal(t, policy.Name, toClient.Name)

	assert.Equal(t, policy.Spec.Action, toClient.Spec.Action)
	assert.Equal(t, policy.Spec.Samples, toClient.Spec.Samples)
	assert.Equal(t, policy.Spec.Order, toClient.Spec.Order)
	assert.Equal(t, policy.Spec.MatchType, toClient.Spec.MatchType)

	assert.Equal(t, policy.Spec.Rules[0].From.Namespaces, toClient.Spec.Rules[0].From.Namespaces)
	assert.Equal(t, policy.Spec.Rules[0].From.NotNamespaces, toClient.Spec.Rules[0].From.NotNamespaces)
	assert.Equal(t, policy.Spec.Rules[0].From.IpBlocks, toClient.Spec.Rules[0].From.IpBlocks)
	assert.Equal(t, policy.Spec.Rules[0].From.NotIpBlocks, toClient.Spec.Rules[0].From.NotIpBlocks)
	assert.Equal(t, policy.Spec.Rules[0].From.Principals, toClient.Spec.Rules[0].From.Principals)
	assert.Equal(t, policy.Spec.Rules[0].From.NotPrincipals, toClient.Spec.Rules[0].From.NotPrincipals)
	assert.Equal(t, policy.Spec.Rules[0].From.Extends[0].Key, toClient.Spec.Rules[0].From.Extends[0].Key)
	assert.Equal(t, policy.Spec.Rules[0].From.Extends[0].Value, toClient.Spec.Rules[0].From.Extends[0].Value)
	assert.Equal(t, policy.Spec.Rules[0].From.NotExtends[0].Key, toClient.Spec.Rules[0].From.NotExtends[0].Key)
	assert.Equal(t, policy.Spec.Rules[0].From.NotExtends[0].Value, toClient.Spec.Rules[0].From.NotExtends[0].Value)

	assert.Equal(t, policy.Spec.Rules[0].When.Key, toClient.Spec.Rules[0].When.Key)
	assert.Equal(t, policy.Spec.Rules[0].When.Values[0].Type, toClient.Spec.Rules[0].When.Values[0].Type)
	assert.Equal(t, policy.Spec.Rules[0].When.Values[0].Value, toClient.Spec.Rules[0].When.Values[0].Value)
	assert.Equal(t, policy.Spec.Rules[0].When.NotValues[0].Type, toClient.Spec.Rules[0].When.NotValues[0].Type)
	assert.Equal(t, policy.Spec.Rules[0].When.NotValues[0].Value, toClient.Spec.Rules[0].When.NotValues[0].Value)
}
