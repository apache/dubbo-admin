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

	"github.com/apache/dubbo-admin/pkg/admin/config"
)

func TestGetDiscoveryPath(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "RightTest1",
			args: args{
				address: "127.0.0.1:0",
			},
			want: "127.0.0.1:" + config.PrometheusMonitorPort,
		},
		{
			name: "RightTest2",
			args: args{
				address: "192.168.127.153",
			},
			want: "192.168.127.153:" + config.PrometheusMonitorPort,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := GetDiscoveryPath(tt.args.address)
			if !reflect.DeepEqual(path, tt.want) {
				t.Errorf("GetDiscoveryPath() = %v, want %v", path, tt.want)
			}
		})
	}
}
