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
	"io/ioutil"
	"log"
	"os"
	"text/template"

	"gopkg.in/yaml.v2"
)

type ConfigData struct {
	ProtoResource string
}

type Resource struct {
	OutPut string   `yaml:"out-put"`
	Name   []string `yaml:"name"`
}

func readYAMLFile(filePath string) (*Resource, error) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config Resource

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func main() {
	config, err := readYAMLFile("tools/deepcopy-gen/metadata.yaml")
	if err != nil {
		panic(err)
	}
	outputFile := config.OutPut
	names := config.Name
	flag.Parse()

	tmpl := template.Must(template.ParseFiles("tools/deepcopy-gen/template.go.tmpl"))

	var typeList []ConfigData

	for _, name := range names {
		data := ConfigData{
			ProtoResource: name,
		}
		typeList = append(typeList, data)
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, typeList); err != nil {
		log.Fatal(err)
	}

	out, err := format.Source(buffer.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	// Output
	if outputFile == "" {
		fmt.Println(string(out))
	} else if err := ioutil.WriteFile(outputFile, out, 0o644); err != nil {
		panic(err)
	}
}
