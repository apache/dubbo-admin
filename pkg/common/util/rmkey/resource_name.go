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

package rmkey

import (
	"strings"

	utilk8s "github.com/apache/dubbo-admin/pkg/common/util/k8s"
)

const (
	firstDelimiter  = "-"
	secondDelimiter = "."
	separator       = "/"
)

func GenerateMetadataResourceKey(app string, revision string, namespace string) string {
	res := app
	if revision != "" {
		res += firstDelimiter + revision
	}
	if namespace != "" {
		res += secondDelimiter + revision
	}
	return res
}

func GenerateNamespacedName(name string, namespace string) string {
	if namespace == "" { // it's cluster scoped object
		return name
	}
	return utilk8s.K8sNamespacedNameToCoreName(name, namespace)
}

func GenerateMappingResourceKey(interfaceName string, namespace string) string {
	res := strings.ToLower(strings.ReplaceAll(interfaceName, ".", "-"))
	if namespace == "" {
		return res
	}
	return utilk8s.K8sNamespacedNameToCoreName(res, namespace)
}
