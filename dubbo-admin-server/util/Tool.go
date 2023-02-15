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
	"strings"
)

func GetInterface(service string) string {
	if len(service) > 0 {
		index := strings.Index(service, "/")
		if index >= 0 {
			service = service[index+1:]
		}
		index = strings.LastIndex(service, ":")
		if index >= 0 {
			service = service[0:index]
		}
	}
	return service
}

func GetGroup(service string) string {
	if len(service) > 0 {
		index := strings.Index(service, "/")
		if index >= 0 {
			return service[0:index]
		}
	}
	return ""
}

func GetVersion(service string) string {
	if len(service) > 0 {
		index := strings.LastIndex(service, ":")
		if index >= 0 {
			return service[index+1:]
		}
	}
	return ""
}
