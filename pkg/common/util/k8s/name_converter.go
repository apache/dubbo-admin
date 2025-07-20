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

package k8s

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

func CoreNameToK8sName(coreName string) (string, string, error) {
	idx := strings.LastIndex(coreName, ".")
	if idx == -1 {
		return "", "", errors.Errorf(`name %q must include namespace after the dot, ex. "name.namespace"`, coreName)
	}
	// namespace cannot contain "." therefore it's always the last part
	namespace := coreName[idx+1:]
	if namespace == "" {
		return "", "", errors.New("namespace must be non-empty")
	}
	return coreName[:idx], namespace, nil
}

func K8sNamespacedNameToCoreName(name, namespace string) string {
	return fmt.Sprintf("%s.%s", name, namespace)
}
