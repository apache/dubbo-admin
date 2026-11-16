/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package monitoring

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// This checks the dashboard contract, not PromQL evaluation against samples.
func TestInstanceMetricQueriesScopeSelectedIP(t *testing.T) {
	content, err := os.ReadFile("grafana-dashboardDefinitions.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Items []struct {
			Data map[string]string `json:"data"`
		} `json:"items"`
	}
	if err := yaml.Unmarshal(content, &config); err != nil {
		t.Fatal(err)
	}
	var dashboard map[string]any
	for _, item := range config.Items {
		for name, data := range item.Data {
			var parsed map[string]any
			if err := json.Unmarshal([]byte(data), &parsed); err != nil {
				t.Fatalf("invalid dashboard JSON %s: %v", name, err)
			}
			if name == "dubbo-instance-metrics.json" {
				dashboard = parsed
			}
		}
	}
	if dashboard == nil {
		t.Fatal("Instance Metric dashboard is missing")
	}

	queries := 0
	failureQueries := map[string]int{
		"dubbo_provider_requests_failed_network_total": 0,
		"dubbo_provider_requests_failed_codec_total":   0,
	}
	var visit func(any)
	visit = func(value any) {
		switch value := value.(type) {
		case map[string]any:
			if expr, ok := value["expr"].(string); ok && expr != "" {
				queries++
				// Inspect every expression, including those that omit $instance entirely.
				for _, required := range []string{
					`application_name="$application",ip="$instance"`,
					`and on(application_name,instance) dubbo_application_info_total{application_name="$application",ip="$instance"}`,
				} {
					if !strings.Contains(expr, required) {
						t.Errorf("Instance query %v lacks %q: %s", value["refId"], required, expr)
					}
				}
				for metric := range failureQueries {
					if !strings.HasPrefix(expr, "sum(") || !strings.Contains(expr, metric+"{") {
						continue
					}
					failureQueries[metric]++
					want := fmt.Sprintf(`sum((increase(%s{application_name="$application",ip="$instance"}[2m]) or (increase(%s{application_name="$application"}[2m]) and on(application_name,instance) dubbo_application_info_total{application_name="$application",ip="$instance"}))) OR vector(0)`, metric, metric)
					if expr != want {
						t.Errorf("Instance error summary for %s: got %s; want %s", metric, expr, want)
					}
				}
			}
			for _, child := range value {
				visit(child)
			}
		case []any:
			for _, child := range value {
				visit(child)
			}
		}
	}
	visit(dashboard)
	if queries == 0 {
		t.Fatal("Instance Metric dashboard has no queries")
	}
	for metric, count := range failureQueries {
		if count != 1 {
			t.Errorf("expected one Instance error summary for %s, got %d", metric, count)
		}
	}
	t.Logf("checked all %d Instance Metric query expressions", queries)
}
