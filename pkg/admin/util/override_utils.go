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

package util

import (
	"net/url"
	"strconv"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
)

func OldOverride2URL(o *model.OldOverride) (*common.URL, error) {
	group := GetGroup(o.Service)
	version := GetVersion(o.Service)
	interfaceName := GetInterface(o.Service)
	var sb strings.Builder
	sb.WriteString(constant.OverrideProtocol)
	sb.WriteString("://")
	if o.Address != "" && o.Address != constant.AnyValue {
		sb.WriteString(o.Address)
	} else {
		sb.WriteString(constant.AnyHostValue)
	}
	sb.WriteString("/")
	sb.WriteString(interfaceName)
	sb.WriteString("?")
	params, _ := url.ParseQuery(o.Params)
	params.Set(constant.CategoryKey, constant.ConfiguratorsCategory)
	params.Set(constant.EnabledKey, strconv.FormatBool(o.Enabled))
	params.Set(constant.DynamicKey, "false")
	if o.Application != "" && o.Application != constant.AnyValue {
		params.Set(constant.ApplicationKey, o.Application)
	}
	if group != "" {
		params.Set(constant.GroupKey, group)
	}
	if version != "" {
		params.Set(constant.VersionKey, version)
	}
	sb.WriteString(params.Encode())

	return common.NewURL(sb.String())
}
