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

package authentication

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopy(t *testing.T) {
	policy := &Policy{
		Name: "test-policy",
		Spec: &PolicySpec{
			Action: "",
			Selector: []*Selector{
				{
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
			},
			PortLevel: []*PortLevel{
				{
					Port:   1314520,
					Action: "test-action",
				},
			},
		},
	}

	toClient := policy.CopyToClient()

	assert.Equal(t, policy.Name, toClient.Name)
	assert.Equal(t, policy.Spec.Action, toClient.Spec.Action)

	assert.Equal(t, policy.Spec.PortLevel[0].Port, toClient.Spec.PortLevel[0].Port)
	assert.Equal(t, policy.Spec.PortLevel[0].Action, toClient.Spec.PortLevel[0].Action)
}
