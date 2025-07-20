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

package util

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	Repositories = "repositories"
)

func Dir() (path string) {
	if home, err := os.UserHomeDir(); err == nil {
		path = filepath.Join(home, ".config", "dubbo")
	}
	return
}

func GetCreatePath() (err error) {
	if err = os.MkdirAll(Dir(), os.ModePerm); err != nil {
		return fmt.Errorf("error creating global config path: %v", err)
	}
	if err = os.MkdirAll(RepositoriesPath(), os.ModePerm); err != nil {
		return fmt.Errorf("error creating global config repositories path: %v", err)
	}
	return
}

func RepositoriesPath() string {
	path := filepath.Join(Dir(), Repositories)
	if e := os.Getenv("DUBBO_REPOSITORIES_PATH"); e != "" {
		path = e
	}
	return path
}
