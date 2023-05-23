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
	"reflect"
	"testing"

	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
)

func TestOldOverride2URL(t *testing.T) {
	type args struct {
		o *model.OldOverride
	}
	tests := []struct {
		name    string
		args    args
		want    *common.URL
		wantErr bool
	}{
		{
			name: "RightTest",
			args: args{
				o: &model.OldOverride{
					Service:     "group/service:1.0.0",
					Address:     "192.168.1.1:8080",
					Enabled:     true,
					Application: "app",
				},
			},
			want: common.NewURLWithOptions(
				common.WithProtocol(constant.OverrideProtocol),
				common.WithIp("192.168.1.1"),
				common.WithPort("8080"),
				common.WithPath("service"),
				common.WithParamsValue(constant.CategoryKey, constant.ConfiguratorsCategory),
				common.WithParamsValue(constant.EnabledKey, "true"),
				common.WithParamsValue(constant.DynamicKey, "false"),
				common.WithParamsValue(constant.ApplicationKey, "app"),
				common.WithParamsValue(constant.GroupKey, "group"),
				common.WithParamsValue(constant.VersionKey, "1.0.0"),
			),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OldOverride2URL(tt.args.o)
			if (err != nil) != tt.wantErr {
				t.Errorf("OldOverride2URL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.String(), tt.want.String()) {
				t.Errorf("OldOverride2URL() = %v, want %v", got, tt.want)
			}
		})
	}
}
