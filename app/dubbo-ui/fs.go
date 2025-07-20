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

package ui

import (
	"embed"
	"io/fs"
)

// By default, go embed does not embed files that starts with `_` that's why we need to use *
// Data Run 'make build-ui' first to generate the distribution of the ui pages.
//
//go:embed dist/*
var Data embed.FS

type tryOrDefault struct {
	fileSystem   fs.FS
	fallbackFile string
}

func (t tryOrDefault) Open(name string) (fs.File, error) {
	f, err := t.fileSystem.Open(name)
	if err == nil {
		return f, nil
	}
	return t.fileSystem.Open(t.fallbackFile)
}

var FS = func() fs.FS {
	fsys, err := fs.Sub(Data, "dist/admin")
	if err != nil {
		panic(err)
	}
	// Adapt Vue's HTML5 history mode
	fsys = tryOrDefault{
		fileSystem:   fsys,
		fallbackFile: "index.html",
	}
	return fsys
}
