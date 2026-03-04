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

package server

// ServerSpec defines server configuration
type ServerSpec struct {
	Port         int      `yaml:"port"`
	Host         string   `yaml:"host"`
	Debug        bool     `yaml:"debug"`
	CORSOrigins  []string `yaml:"cors_origins"`
	ReadTimeout  int      `yaml:"read_timeout"`
	WriteTimeout int      `yaml:"write_timeout"`
}

// DefaultServerSpec returns default server configuration
func DefaultServerSpec() *ServerSpec {
	return &ServerSpec{
		Port:         8888,
		Host:         "0.0.0.0",
		Debug:        false,
		CORSOrigins:  []string{"*"},
		ReadTimeout:  30,
		WriteTimeout: 30,
	}
}
