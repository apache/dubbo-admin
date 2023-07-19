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

package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	prom_v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

func stitchingLabels(labels []string) string {
	var labelsQ string
	for i, labelsInstance := range labels {
		if i == 0 {
			labelsQ += labelsInstance
		} else {
			labelsQ += ", " + labelsInstance
		}
	}
	return labelsQ
}

func FetchQuery(ctx context.Context, api prom_v1.API, metricName string, labels []string) Metric {
	var query string
	// Example: sum(my_counter{name=dubbo})
	label := stitchingLabels(labels)
	query = fmt.Sprintf("sum(%s{%s})", metricName, label)
	logger.Sugar().Info(query)
	result, warnings, err := api.Query(ctx, query, time.Now())
	switch result.Type() {
	case model.ValVector:
		return Metric{Vector: result.(model.Vector)}
	}
	if len(warnings) > 0 {
		logger.Sugar().Warnf("Warnings: %v", warnings)
	}
	if err != nil {
		logger.Sugar().Errorf("Error query Prometheus: %v\n", err)
		return Metric{Err: fmt.Errorf("Error query Prometheus: %v\n", err)}
	}
	return Metric{Err: fmt.Errorf("invalid query, matrix expected: %s", query)}
}
