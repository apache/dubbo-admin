// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//

//	http://www.apache.org/licenses/LICENSE-2.0

//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"sync"
	"testing"

	"github.com/apache/dubbo-admin/pkg/admin/config"

	"github.com/apache/dubbo-admin/pkg/admin/cache"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"

	"dubbo.apache.org/dubbo-go/v3/common"
)

var prometheusService MonitorService = &PrometheusServiceImpl{}

type args struct {
	address []string
}

type test struct {
	name    string
	args    args
	want    []model.Target
	wantErr error
}

func initCache(test []test) {
	proService := &sync.Map{}
	conService := &sync.Map{}
	// protest1
	protest1QueryParams := url.Values{
		constant.ApplicationKey: {"protest1QueryParams"},
	}
	protest1, _ := common.NewURL(test[0].args.address[0],
		common.WithProtocol(constant.AdminProtocol),
		common.WithParams(protest1QueryParams),
		common.WithLocation(test[0].args.address[0]),
	)
	// protest2
	protest2QueryParams := url.Values{
		constant.ApplicationKey: {"protest2QueryParams"},
	}
	protest2, _ := common.NewURL(test[0].args.address[1],
		common.WithProtocol(constant.AdminProtocol),
		common.WithParams(protest2QueryParams),
		common.WithLocation(test[0].args.address[1]),
	)

	contest1QueryParams := url.Values{
		constant.ApplicationKey: {"protest1QueryParams"},
	}
	// consumer test1
	contest1, _ := common.NewURL(test[0].args.address[2],
		common.WithProtocol(constant.AdminProtocol),
		common.WithParams(contest1QueryParams),
		common.WithLocation(test[0].args.address[2]),
	)
	// consumer test2
	contest2QueryParams := url.Values{
		constant.ApplicationKey: {"protest2QueryParams"},
	}
	contest2, _ := common.NewURL(test[0].args.address[3],
		common.WithProtocol(constant.AdminProtocol),
		common.WithParams(contest2QueryParams),
		common.WithLocation(test[0].args.address[3]),
	)
	proService.Store("providers", map[string]*common.URL{
		"protest1": protest1,
		"protest2": protest2,
	})

	conService.Store("consumers", map[string]*common.URL{
		"contest1": contest1,
		"contest2": contest2,
	})

	cache.InterfaceRegistryCache.Store(constant.ProvidersCategory, proService)
	cache.InterfaceRegistryCache.Store(constant.ConsumersCategory, conService)
}

// Simulate Prometheus to send requests for http_sd service discovery.
func initPromClient(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// Simulate Prometheus to periodically send requests to admin to realize http_ds service discovery.
func TestPrometheusServiceImpl_PromDiscovery(t *testing.T) {
	tests := []test{
		{
			name: "TEST",
			args: args{
				address: []string{
					"127.0.0.1:0",
					"198.127.163.150:8080",
					"198.127.163.153:0",
					"198.127.163.151:0",
				},
			},
			wantErr: nil,
			want: []model.Target{
				{
					Labels: map[string]string{},
					Targets: []string{
						"127.0.0.1:" + config.PrometheusMonitorPort,
						"198.127.163.150:" + config.PrometheusMonitorPort,
						"198.127.163.153:" + config.PrometheusMonitorPort,
						"198.127.163.151:" + config.PrometheusMonitorPort,
					},
				},
			},
		},
	}
	initCache(tests)
	defer cache.InterfaceRegistryCache.Delete(constant.ProvidersCategory)
	defer cache.InterfaceRegistryCache.Delete(constant.ConsumersCategory)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				target, err := prometheusService.PromDiscovery(w)
				if err != nil {
					t.Errorf("Server Start Error: %v\n", err)
				}
				for i := 0; i < len(target); i++ {
					gots := target[i].Targets
					targets := tt.want[i].Targets
					sort.Strings(gots)
					sort.Strings(targets)
					target[i].Targets = gots
					tt.want[i].Targets = targets
				}
				if !reflect.DeepEqual(target, tt.want) {
					t.Errorf("PromDiscovery() got = %v, want %v", target, tt.want)
				}
			}))
			defer ts.Close()
			addr := ts.URL
			_, err := initPromClient(addr)
			if err != nil {
				t.Errorf("Error: %v\n", err)
			}
		})
	}
}
