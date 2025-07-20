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

package files

import (
	"go/build"
	"os"
	"path"
	"strings"
)

func GetProjectRoot(file string) string {
	dir := file
	for path.Base(dir) != "pkg" && path.Base(dir) != "app" {
		dir = path.Dir(dir)
	}
	return path.Dir(dir)
}

func GetProjectRootParent(file string) string {
	return path.Dir(GetProjectRoot(file))
}

func RelativeToPkgMod(file string) string {
	root := path.Dir(path.Dir(path.Dir(GetProjectRoot(file))))
	return strings.TrimPrefix(file, root)
}

func RelativeToProjectRoot(path string) string {
	root := GetProjectRoot(path)
	return strings.TrimPrefix(path, root)
}

func RelativeToProjectRootParent(path string) string {
	root := GetProjectRootParent(path)
	return strings.TrimPrefix(path, root)
}

func GetGopath() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	return gopath
}
