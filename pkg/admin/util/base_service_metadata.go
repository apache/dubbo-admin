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
	"bytes"
	"strings"

	"github.com/apache/dubbo-admin/pkg/admin/constant"
)

func BuildServiceKey(app, service, version, group string) string {
	if app != "" {
		return app
	}
	// id format: "${class}:${version}:${group}"
	return service + constant.Colon + version + constant.Colon + group
}

func ServiceKey(intf string, group string, version string) string {
	if intf == "" {
		return ""
	}
	buf := &bytes.Buffer{}
	if group != "" {
		buf.WriteString(group)
		buf.WriteString("/")
	}

	buf.WriteString(intf)

	if version != "" && version != "0.0.0" {
		buf.WriteString(":")
		buf.WriteString(version)
	}

	return buf.String()
}

func ColonSeparatedKey(intf string, group string, version string) string {
	if intf == "" {
		return ""
	}
	var buf strings.Builder
	buf.WriteString(intf)
	buf.WriteString(":")
	if version != "" && version != "0.0.0" {
		buf.WriteString(version)
	}
	buf.WriteString(":")
	if group != "" {
		buf.WriteString(group)
	}
	return buf.String()
}
