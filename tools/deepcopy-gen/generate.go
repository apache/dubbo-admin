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
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"text/template"
)

type ConfigData struct {
	ProtoResource string
}

type Resource struct {
	OutPut string   `yaml:"out-put"`
	Name   []string `yaml:"name"`
}

func readYAMLFile(filePath string) (*Resource, error) {
	// 读取YAML文件内容
	yamlFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// 定义一个变量用于存储解析后的数据
	var config Resource

	// 解析YAML文件内容
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
	} else if err := ioutil.WriteFile(outputFile, out, 0644); err != nil {
		panic(err)
	}
}
