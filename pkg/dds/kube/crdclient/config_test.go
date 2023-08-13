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

package crdclient

import (
	"reflect"
	"testing"

	"github.com/apache/dubbo-admin/pkg/core/model"
	"github.com/apache/dubbo-admin/pkg/core/schema/collection"
	"github.com/apache/dubbo-admin/pkg/core/schema/resource"
	"github.com/davecgh/go-spew/spew"
)

// getByMessageName finds a schema by message name if it is available
// In test setup, we do not have more than one descriptor with the same message type, so this
// function is ok for testing purpose.
func getByMessageName(schemas collection.Schemas, name string) (collection.Schema, bool) {
	for _, s := range schemas.All() {
		if s.Resource().Proto() == name {
			return s, true
		}
	}
	return nil, false
}

func schemaFor(kind, proto string) collection.Schema {
	return collection.Builder{
		Name: kind,
		Resource: resource.Builder{
			Kind:   kind,
			Plural: kind + "s",
			Proto:  proto,
		}.BuildNoValidate(),
	}.MustBuild()
}

func TestConfigDescriptor(t *testing.T) {
	a := schemaFor("a", "proxy.A")
	schemas := collection.SchemasFor(
		a,
		schemaFor("b", "proxy.B"),
		schemaFor("c", "proxy.C"))
	want := []string{"a", "b", "c"}
	got := schemas.Kinds()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("descriptor.Types() => got %+vwant %+v", spew.Sdump(got), spew.Sdump(want))
	}

	aType, aExists := schemas.FindByGroupVersionKind(a.Resource().GroupVersionKind())
	if !aExists || !reflect.DeepEqual(aType, a) {
		t.Errorf("descriptor.GetByType(a) => got %+v, want %+v", aType, a)
	}
	if _, exists := schemas.FindByGroupVersionKind(model.GroupVersionKind{Kind: "missing"}); exists {
		t.Error("descriptor.GetByType(missing) => got true, want false")
	}

	aSchema, aSchemaExists := getByMessageName(schemas, a.Resource().Proto())
	if !aSchemaExists || !reflect.DeepEqual(aSchema, a) {
		t.Errorf("descriptor.GetByMessageName(a) => got %+v, want %+v", aType, a)
	}
	_, aSchemaNotExist := getByMessageName(schemas, "blah")
	if aSchemaNotExist {
		t.Errorf("descriptor.GetByMessageName(blah) => got true, want false")
	}
}

func TestEventString(t *testing.T) {
	cases := []struct {
		in   Event
		want string
	}{
		{EventAdd, "add"},
		{EventUpdate, "update"},
		{EventDelete, "delete"},
	}
	for _, c := range cases {
		if got := c.in.String(); got != c.want {
			t.Errorf("Failed: got %q want %q", got, c.want)
		}
	}
}
