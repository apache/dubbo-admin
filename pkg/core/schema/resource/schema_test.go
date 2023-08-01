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

package resource

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/types"
	. "github.com/onsi/gomega"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name        string
		b           Builder
		expectError bool
	}{
		{
			name: "valid",
			b: Builder{
				Kind:   "Empty",
				Plural: "Empties",
				Proto:  "google.protobuf.Empty",
			},
			expectError: false,
		},
		{
			name: "invalid kind",
			b: Builder{
				Kind:   "",
				Plural: "Empties",
				Proto:  "google.protobuf.Empty",
			},
			expectError: true,
		},
		{
			name: "invalid plural",
			b: Builder{
				Kind:   "Empty",
				Plural: "",
				Proto:  "google.protobuf.Empty",
			},
			expectError: true,
		},
		{
			name: "invalid proto",
			b: Builder{
				Kind:   "Boo",
				Plural: "Boos",
				Proto:  "boo",
			},
			expectError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := NewWithT(t)

			err := c.b.BuildNoValidate().Validate()
			if c.expectError {
				g.Expect(err).ToNot(BeNil())
			} else {
				g.Expect(err).To(BeNil())
			}
		})
	}
}

func TestBuild(t *testing.T) {
	cases := []struct {
		name        string
		b           Builder
		expectError bool
	}{
		{
			name: "valid",
			b: Builder{
				Kind:   "Empty",
				Plural: "Empties",
				Proto:  "google.protobuf.Empty",
			},
			expectError: false,
		},
		{
			name: "invalid kind",
			b: Builder{
				Kind:   "",
				Plural: "Empties",
				Proto:  "google.protobuf.Empty",
			},
			expectError: true,
		},
		{
			name: "invalid plural",
			b: Builder{
				Kind:   "Empty",
				Plural: "",
				Proto:  "google.protobuf.Empty",
			},
			expectError: true,
		},
		{
			name: "invalid proto",
			b: Builder{
				Kind:   "Boo",
				Plural: "Boos",
				Proto:  "boo",
			},
			expectError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := NewWithT(t)

			_, err := c.b.Build()
			if c.expectError {
				g.Expect(err).ToNot(BeNil())
			} else {
				g.Expect(err).To(BeNil())
			}
		})
	}
}

func TestCanonicalName(t *testing.T) {
	cases := []struct {
		name     string
		s        Schema
		expected string
	}{
		{
			name: "group",
			s: Builder{
				Group:   "g",
				Version: "v",
				Kind:    "k",
				Plural:  "ks",
				Proto:   "google.protobuf.Empty",
			}.MustBuild(),
			expected: "g/v/k",
		},
		{
			name: "no group",
			s: Builder{
				Group:   "",
				Version: "v",
				Kind:    "k",
				Plural:  "ks",
				Proto:   "google.protobuf.Empty",
			}.MustBuild(),
			expected: "core/v/k",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(c.s.GroupVersionKind().String()).To(Equal(c.expected))
		})
	}
}

func TestNewProtoInstance(t *testing.T) {
	g := NewWithT(t)

	s := Builder{
		Kind:   "Empty",
		Plural: "Empties",
		Proto:  "google.protobuf.Empty",
	}.MustBuild()

	p, err := s.NewInstance()
	g.Expect(err).To(BeNil())
	g.Expect(p).To(Equal(&types.Empty{}))
}

func TestMustNewProtoInstance_Panic_Nil(t *testing.T) {
	g := NewWithT(t)
	defer func() {
		r := recover()
		g.Expect(r).NotTo(BeNil())
	}()
	old := protoMessageType
	defer func() {
		protoMessageType = old
	}()
	protoMessageType = func(name string) reflect.Type {
		return nil
	}

	s := Builder{
		Kind:  "Empty",
		Proto: "google.protobuf.Empty",
	}.MustBuild()

	_ = s.MustNewInstance()
}
