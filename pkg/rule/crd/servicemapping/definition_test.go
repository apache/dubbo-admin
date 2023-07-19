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

package servicemapping

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopy(t *testing.T) {
	policy := &Policy{
		Name: "test-policy",
		Spec: &PolicySpec{
			InterfaceName:    "test-interface",
			ApplicationNames: []string{"test-application"},
		},
	}

	toClient := policy.CopyToClient()

	assert.Equal(t, policy.Name, toClient.Name)
	assert.Equal(t, policy.Spec.InterfaceName, toClient.Spec.InterfaceName)
	assert.Equal(t, policy.Spec.ApplicationNames[0], toClient.Spec.ApplicationNames[0])
}
