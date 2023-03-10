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

package prometheus

import (
	"encoding/json"
	"fmt"
	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"log"
	"net/http"
	"time"
)

// GetPromResult Parse the data read from Prometheus
func GetPromResult(url string, result interface{}) error {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	r, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	err = json.NewDecoder(r.Body).Decode(result)
	if err != nil {
		return err
	}
	return nil
}

func FetchRadio(metricName1 string, metricName2 string, label1 []string, label2 []string) (*QueryInfo, error) {
	var query1 string
	var query2 string
	var query string
	// Example: sum(http_success{}) / sum(http_total{})
	var labelsQ1 string
	for i, labelsInstance := range label1 {
		if i == 0 {
			labelsQ1 += labelsInstance
		} else {
			labelsQ1 += ", " + labelsInstance
		}
	}
	query1 = fmt.Sprintf("sum(%s{%s})", metricName1, labelsQ1)
	var labelsQ2 string
	for i, labelsInstance := range label2 {
		if i == 0 {
			labelsQ2 += labelsInstance
		} else {
			labelsQ2 += ", " + labelsInstance
		}
	}
	query2 = fmt.Sprintf("sum(%s{%s})", metricName2, labelsQ2)
	query = fmt.Sprintf("%s/%s", query1, query2)
	prometheusUrl := fmt.Sprintf("http://%s:%s", config.PrometheusIp, config.PrometheusPort)
	ustr := prometheusUrl + constant.EpQuery + query
	log.Println(ustr)
	info := &QueryInfo{}
	err := GetPromResult(ustr, info)
	if err != nil {
		panic(err)
	}
	return info, nil
}

// FetchRange Grab the range data of Rate type
// metricName: Indicator name
// labels:
func FetchRange(metricName string, labels []string) (*QueryInfo, error) {
	var query string
	// Example: sum(my_counter{})
	var labelsQ string
	for i, labelsInstance := range labels {
		if i == 0 {
			labelsQ += labelsInstance
		} else {
			labelsQ += ", " + labelsInstance
		}
	}
	query = fmt.Sprintf("sum(%s{%s})", metricName, labelsQ)
	prometheusUrl := fmt.Sprintf("http://%s:%s", config.PrometheusIp, config.PrometheusPort)
	ustr := prometheusUrl + constant.EpQuery + query
	log.Println(ustr)
	info := &QueryInfo{}

	err := GetPromResult(ustr, info)
	if err != nil {
		return info, err
	}
	return info, nil
}
