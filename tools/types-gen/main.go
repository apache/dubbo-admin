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

package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"text/template"

	"github.com/apache/dubbo-admin/pkg/core/schema/collection"
	"github.com/apache/dubbo-admin/pkg/core/schema/collections"
)

// ConfigData is data struct to feed to types.go template
type ConfigData struct {
	Namespaced      bool
	VariableName    string
	APIImport       string
	ClientImport    string
	Kind            string
	ClientGroupPath string
	ClientTypePath  string

	Client     string
	TypeSuffix string
}

// MakeConfigData prepare data for code generation for the given schema.
func MakeConfigData(schema collection.Schema) ConfigData {
	out := ConfigData{
		Namespaced:      !schema.Resource().IsClusterScoped(),
		VariableName:    schema.VariableName(),
		APIImport:       "dubbo_apache_org_v1alpha1",
		Kind:            schema.Resource().Kind(),
		ClientImport:    "v1alpha1",
		ClientTypePath:  clientGoTypePath[schema.Resource().Plural()],
		ClientGroupPath: "DubboV1alpha1",
		Client:          "ic",
		TypeSuffix:      "",
	}
	log.Printf("Generating Dubbo type %s for %s/%s CRD\n", out.VariableName, out.APIImport, out.Kind)
	return out
}

// Translates a plural type name to the type path in client-go
// TODO: can we automatically derive this? I don't think we can, its internal to the kubegen
var clientGoTypePath = map[string]string{
	"authenticationpolicies": "AuthenticationPolicies",
	"authorizationpolicies":  "AuthorizationPolicies",
	"conditionroutes":        "ConditionRoutes",
	"tagroutes":              "TagRoutes",
	"dynamicconfigs":         "DynamicConfigs",
	"servicenamemappings":    "ServiceNameMappings",
}

func main() {
	tempateFile := flag.String("template", "./types.go.tmpl", "Template file")
	outputFile := flag.String("output", "../../pkg/dds/kube/crdclient/types.gen.go", "Output file. Leave blank to go to stdout")
	flag.Parse()

	tmpl := template.Must(template.ParseFiles(*tempateFile))

	var typeList []ConfigData
	for _, s := range collections.Rule.All() {
		typeList = append(typeList, MakeConfigData(s))
	}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, typeList); err != nil {
		log.Fatal(fmt.Errorf("template: %v", err))
	}

	// Format source code.
	out, err := format.Source(buffer.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	// Output
	if *outputFile == "" || outputFile == nil {
		fmt.Println(string(out))
	} else {
		file, err := os.Create(*outputFile)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		_, err = file.Write(out)
		if err != nil {
			panic(err)
		}
	}
}
