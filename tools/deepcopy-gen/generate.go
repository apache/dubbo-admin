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

	"github.com/apache/dubbo-admin/pkg/core/schema/collections"
)

type ConfigData struct {
	Kind string
}

func main() {
	output := flag.String("output", "../../api/resource/v1alpha1/resource_deepcopy.go", "output Path")
	tem := flag.String("template", "./template.go.tmpl", "Template file")
	flag.Parse()

	var kindList []ConfigData

	for _, s := range collections.Rule.All() {
		kindList = append(kindList, ConfigData{Kind: s.Resource().Kind()})
	}

	tmpl := template.Must(template.ParseFiles(*tem))
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, kindList); err != nil {
		log.Fatal(fmt.Errorf("template: %v", err))
	}

	out, err := format.Source(buffer.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	if output == nil || *output == "" {
		fmt.Println(output)
	}
	file, err := os.Create(*output)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = file.Write(out)
	if err != nil {
		panic(err)
	}
}
