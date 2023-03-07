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

package config

import (
	"flag"
	"os"
	"strconv"
)

func GetOptions() *Options {
	// TODO read options from env
	options := Options{
		Namespace:   GetStringEnv("namespace", "dubbo-system"),
		ServiceName: GetStringEnv("servicename", "dubbo-ca"),

		PlainServerPort:  GetIntEnv("plainserverport", 30060),
		SecureServerPort: GetIntEnv("secureserverport", 30062),
		DebugPort:        GetIntEnv("debugport", 30070),

		WebhookPort:       30080,
		WebhookAllowOnErr: GetBoolEnv("webhookallowonerr", true),

		CaValidity:   30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity: 1 * 60 * 60 * 1000,       // 1 hour

		InPodEnv:              GetBoolEnv("inpodenv", false),
		IsKubernetesConnected: GetBoolEnv("iskubernetesconnected", false),
		EnableOIDCCheck:       GetBoolEnv("enableoidccheck", true),
	}

	flag.StringVar(&options.Namespace, "namespace", options.Namespace, "dubbo namespace")
	flag.StringVar(&options.ServiceName, "servicename", options.ServiceName, "dubbo serviceName")
	flag.IntVar(&options.PlainServerPort, "plainserverport", options.PlainServerPort, "dubbo plainServerPort")
	flag.IntVar(&options.SecureServerPort, "secureserverport", options.SecureServerPort, "dubbo secureServerPort")
	flag.IntVar(&options.DebugPort, "debugport", options.DebugPort, "dubbo debugPort")
	webhookport := GetIntEnv("webhookport", 30080)
	flag.IntVar(&webhookport, "webhookport", webhookport, "dubbo webhookPort")
	flag.BoolVar(&options.WebhookAllowOnErr, "webhookallowonerr", options.WebhookAllowOnErr, "dubbo webhookAllowOnErr")
	flag.BoolVar(&options.InPodEnv, "inpodenv", options.InPodEnv, "dubbo inPodEnv")
	flag.BoolVar(&options.IsKubernetesConnected, "iskubernetesconnected", options.IsKubernetesConnected, "dubbo isKubernetesConnected")
	flag.BoolVar(&options.EnableOIDCCheck, "enableoidccheck", options.EnableOIDCCheck, "dubbo enableOIDCCheck")
	flag.Parse()

	options.WebhookPort = int32(webhookport)
	return &options
}

func GetStringEnv(name string, defvalue string) string {
	val, ex := os.LookupEnv(name)
	if ex {
		return val
	} else {
		return defvalue
	}
}

func GetIntEnv(name string, defvalue int) int {
	val, ex := os.LookupEnv(name)
	if ex {
		num, err := strconv.Atoi(val)
		if err != nil {
			return defvalue
		} else {
			return num
		}
	} else {
		return defvalue
	}
}

func GetBoolEnv(name string, defvalue bool) bool {
	val, ex := os.LookupEnv(name)
	if ex {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return defvalue
		} else {
			return boolVal
		}
	} else {
		return defvalue
	}
}
