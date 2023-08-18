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
	"fmt"
	"os"

	"github.com/apache/dubbo-admin/pkg/core/schema"
	"github.com/apache/dubbo-admin/pkg/core/schema/ast"
	"github.com/apache/dubbo-admin/tools/resource-gen"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Printf("Invalid args: %v", os.Args)
		os.Exit(-1)
	}
	pkg := os.Args[1]
	input := os.Args[2]
	output := os.Args[3]

	// Read the input file
	b, err := os.ReadFile(input)
	if err != nil {
		fmt.Printf("unable to read input file: %v", err)
		os.Exit(-2)
	}

	// Parse the file.
	m, err := ast.Parse(string(b))
	if err != nil {
		fmt.Printf("failed parsing input file: %v", err)
		os.Exit(-3)
	}

	// Validate the input.
	if _, err := schema.Build(m); err != nil {
		fmt.Printf("failed building metadata: %v", err)
		os.Exit(-4)
	}

	if pkg == "gvk" {
		contents, err := resource.WriteGvk(m)
		if err != nil {
			fmt.Printf("Error applying static init template: %v", err)
			os.Exit(-3)
		}
		if err = os.WriteFile(output, []byte(contents), os.ModePerm); err != nil {
			fmt.Printf("Error writing output file: %v", err)
			os.Exit(-4)
		}
		return
	} else {
		contents, err := resource.StaticCollections(m)
		if err != nil {
			fmt.Printf("Error applying static init template: %v", err)
			os.Exit(-3)
		}
		if err = os.WriteFile(output, []byte(contents), os.ModePerm); err != nil {
			fmt.Printf("Error writing output file: %v", err)
			os.Exit(-4)
		}
	}
}
