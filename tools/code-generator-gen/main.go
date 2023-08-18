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
	outputFiletypes := flag.String("ot", "../../pkg/core/gen/apis/dubbo.apache.org/v1alpha1/types.go", "gen types.go")
	outputFileregister := flag.String("or", "../../pkg/core/gen/apis/dubbo.apache.org/v1alpha1/register.go", "gen register.go")
	templateType := flag.String("tt", "./typesgen.go.tmpl", "type.go Template")
	templateRegister := flag.String("tr", "./register.go.tmpl", "register.go Template")
	flag.Parse()

	var kindList []ConfigData

	for _, s := range collections.Rule.All() {
		kindList = append(kindList, ConfigData{Kind: s.Resource().Kind()})
	}

	tmpltypes := template.Must(template.ParseFiles(*templateType))
	tmplregister := template.Must(template.ParseFiles(*templateRegister))
	var buffertypes bytes.Buffer
	if err := tmpltypes.Execute(&buffertypes, kindList); err != nil {
		log.Fatal(fmt.Errorf("template: %v", err))
	}

	var bufferregister bytes.Buffer
	if err := tmplregister.Execute(&bufferregister, kindList); err != nil {
		log.Fatal(fmt.Errorf("template: %v", err))
	}

	outtypes, err := format.Source(buffertypes.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	outregister, err := format.Source(bufferregister.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	if outputFiletypes == nil || *outputFiletypes == "" {
		fmt.Println(outputFiletypes)
	}
	file, err := os.Create(*outputFiletypes)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = file.Write(outtypes)
	if err != nil {
		panic(err)
	}

	if outputFileregister == nil || *outputFileregister == "" {
		fmt.Println(outputFileregister)
	}
	file, err = os.Create(*outputFileregister)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = file.Write(outregister)
	if err != nil {
		panic(err)
	}
}
