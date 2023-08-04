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

// ServiceDTO is the transforming format of service
type ServiceDTO struct {
	Service        string `json:"service"`
	AppName        string `json:"appName"`
	Group          string `json:"group"`
	Version        string `json:"version"`
	RegistrySource string `json:"registrySource"`
}

type ListServiceByPage struct {
	Content       []*ServiceDTO `json:"content"`
	TotalPages    int           `json:"totalPages"`
	TotalElements int           `json:"totalElements"`
	Size          string        `json:"size"`
	First         bool          `json:"first"`
	Last          bool          `json:"last"`
	PageNumber    string        `json:"pageNumber"`
	Offset        int           `json:"offset"`
}

type ServiceTest struct {
	Service        string        `json:"service"`
	Method         string        `json:"method"`
	ParameterTypes []string      `json:"ParameterTypes"`
	Params         []interface{} `json:"params"`
}

type MethodMetadata struct {
	Signature      string        `json:"signature"`
	ParameterTypes []interface{} `json:"parameterTypes"`
	ReturnType     string        `json:"returnType"`
}
