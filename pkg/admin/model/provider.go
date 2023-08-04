// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import "time"

type Provider struct {
	Entity
	Service        string        `json:"service"`
	URL            string        `json:"url"`
	Parameters     string        `json:"parameters"`
	Address        string        `json:"address"`
	Registry       string        `json:"registry"`
	Dynamic        bool          `json:"dynamic"`
	Enabled        bool          `json:"enabled"`
	Timeout        int64         `json:"timeout"`
	Serialization  string        `json:"serialization"`
	Weight         int64         `json:"weight"`
	Application    string        `json:"application"`
	Username       string        `json:"username"`
	Expired        time.Duration `json:"expired"`
	Alived         int64         `json:"alived"`
	RegistrySource string        `json:"registrySource"`
}
