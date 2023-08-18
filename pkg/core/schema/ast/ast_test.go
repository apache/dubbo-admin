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

package ast

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestParse(t *testing.T) {
	cases := []struct {
		input    string
		expected *Metadata
	}{
		{
			input:    ``,
			expected: &Metadata{},
		},
		{
			input: `
resources:
  - kind: "AuthenticationPolicy"
    group: "dubbo.apache.org"
    version: "v1alpha1"
    validate: "EmptyValidate"
    proto: "dubbo.apache.org.v1alpha1.AuthenticationPolicy"
`,
			expected: &Metadata{
				Resources: []*Resource{
					{
						Kind:     "AuthenticationPolicy",
						Group:    "dubbo.apache.org",
						Version:  "v1alpha1",
						Proto:    "dubbo.apache.org.v1alpha1.AuthenticationPolicy",
						Validate: "EmptyValidate",
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			g := NewWithT(t)
			actual, err := Parse(c.input)
			g.Expect(err).To(BeNil())
			g.Expect(actual).To(Equal(c.expected))
		})
	}
}
