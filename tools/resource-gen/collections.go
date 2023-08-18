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
	"fmt"
	"sort"
	"strings"

	"github.com/apache/dubbo-admin/pkg/core/schema/ast"
)

const staticResourceTemplate = `
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

package gvk

import "github.com/apache/dubbo-admin/pkg/core/model"

var (
{{- range .Entries }}
	{{.Type}} = model.GroupVersionKind{Group: "{{.Resource.Group}}", Version: "{{.Resource.Version}}", Kind: "{{.Resource.Kind}}"}.String()
{{- end }}
)
`

const staticCollectionsTemplate = `
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

package collections

import (
	"reflect"

	api "github.com/apache/dubbo-admin/api/resource/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/schema/collection"
	"github.com/apache/dubbo-admin/pkg/core/schema/resource"
	"github.com/apache/dubbo-admin/pkg/core/validation"
)

var (
{{ range .Entries }}
	{{ .Collection.VariableName }} = collection.Builder {
		Name: "{{ .Collection.Name }}",
		VariableName: "{{ .Collection.VariableName }}",
		Resource: resource.Builder {
			Group: "{{ .Resource.Group }}",
			Kind: "{{ .Resource.Kind }}",
			Plural: "{{ .Resource.Plural }}",
			Version: "{{ .Resource.Version }}",
			Proto: "{{ .Resource.Proto }}",
			ReflectType: {{ .Type }},
			ClusterScoped: {{ .Resource.ClusterScoped }},
			ValidateProto: validation.{{ .Resource.Validate }},
		}.MustBuild(),
	}.MustBuild()
{{ end }}

	Rule = collection.NewSchemasBuilder().
	{{- range .Entries }}
		{{- if .Collection.Dds }}
		MustAdd({{ .Collection.VariableName }}).
		{{- end}}
	{{- end }}
		Build()
)
`

type colEntry struct {
	Collection *ast.Collection
	Resource   *ast.Resource
	Type       string
}

func WriteGvk(m *ast.Metadata) (string, error) {
	entries := make([]colEntry, 0, len(m.Collections))
	for _, c := range m.Collections {
		// Filter out Dds ones, as these are duplicated
		if !c.Dds {
			continue
		}
		r := m.FindResourceForGroupKind(c.Group, c.Kind)
		if r == nil {
			return "", fmt.Errorf("failed to find resource (%s/%s) for collection %s", c.Group, c.Kind, c.Name)
		}

		name := r.Kind
		entries = append(entries, colEntry{
			Type:     name,
			Resource: r,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.Compare(entries[i].Type, entries[j].Type) < 0
	})

	context := struct {
		Entries []colEntry
	}{
		Entries: entries,
	}

	return applyTemplate(staticResourceTemplate, context)
}

// StaticCollections generates a Go file for static-importing Proto packages, so that they get registered statically.
func StaticCollections(m *ast.Metadata) (string, error) {
	entries := make([]colEntry, 0, len(m.Collections))
	for _, c := range m.Collections {
		r := m.FindResourceForGroupKind(c.Group, c.Kind)
		if r == nil {
			return "", fmt.Errorf("failed to find resource (%s/%s) for collection %s", c.Group, c.Kind, c.Name)
		}
		spl := strings.Split(r.Proto, ".")
		tname := spl[len(spl)-1]
		e := colEntry{
			Collection: c,
			Resource:   r,
			Type:       fmt.Sprintf("reflect.TypeOf(&api.%s{}).Elem()", tname),
		}
		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.Compare(entries[i].Collection.Name, entries[j].Collection.Name) < 0
	})

	context := struct {
		Entries []colEntry
	}{
		Entries: entries,
	}

	return applyTemplate(staticCollectionsTemplate, context)
}
