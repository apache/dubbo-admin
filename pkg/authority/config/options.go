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
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/pflag"
)

type Options struct {
	Namespace   string
	ServiceName string

	PlainServerPort  int
	SecureServerPort int
	DebugPort        int

	WebhookPort       int32
	WebhookAllowOnErr bool

	CaValidity   int64
	CertValidity int64

	InPodEnv              bool
	IsKubernetesConnected bool
	IsTrustAnyone         bool

	// TODO remove EnableOIDCCheck
	EnableOIDCCheck      bool
	ResourcelockIdentity string

	// Qps for rest config
	RestConfigQps int
	// Burst for rest config
	RestConfigBurst int
}

func NewOptions() *Options {
	return &Options{
		Namespace:         "dubbo-system",
		ServiceName:       "dubbo-ca",
		PlainServerPort:   30060,
		SecureServerPort:  30062,
		DebugPort:         30070,
		WebhookPort:       30080,
		WebhookAllowOnErr: true,
		CaValidity:        30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:      1 * 60 * 60 * 1000,       // 1 hour

		InPodEnv:              false,
		IsKubernetesConnected: false,
		EnableOIDCCheck:       true,
		ResourcelockIdentity:  GetStringEnv("POD_NAME", GetDefaultResourcelockIdentity()),
		RestConfigQps:         50,
		RestConfigBurst:       100,
	}
}

func (o *Options) FillFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.Namespace, "namespace", "dubbo-system", "dubbo namespace")
	flags.StringVar(&o.ServiceName, "service-name", "dubbo-ca", "dubbo service name")
	flags.IntVar(&o.PlainServerPort, "plain-server-port", 30060, "dubbo plain server port")
	flags.IntVar(&o.SecureServerPort, "secure-server-port", 30062, "dubbo secure server port")
	flags.IntVar(&o.DebugPort, "debug-port", 30070, "dubbo debug port")
	flags.Int32Var(&o.WebhookPort, "webhook-port", 30080, "dubbo webhook port")
	flags.BoolVar(&o.WebhookAllowOnErr, "webhook-allow-on-err", true, "dubbo webhook allow on error")
	flags.BoolVar(&o.InPodEnv, "in-pod-env", false, "dubbo run in pod environment")
	flags.BoolVar(&o.IsKubernetesConnected, "is-kubernetes-connected", false, "dubbo connected with kubernetes")
	flags.BoolVar(&o.EnableOIDCCheck, "enable-oidc-check", false, "dubbo enable OIDC check")
	flags.IntVar(&o.RestConfigQps, "rest-config-qps", 50, "qps for rest config")
	flags.IntVar(&o.RestConfigBurst, "rest-config-burst", 100, "burst for rest config")
}

func (o *Options) Validate() []error {
	// TODO validate options
	return nil
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

func GetDefaultResourcelockIdentity() string {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	randomBytes := make([]byte, 5)
	_, err = rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	randomStr := base32.StdEncoding.EncodeToString(randomBytes)
	return fmt.Sprintf("%s-%s", hostname, randomStr)
}
