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

package security

type SecurityConfig struct {
	CaValidity           int64  `yaml:"ca-validity"`
	CertValidity         int64  `yaml:"cert-validity"`
	IsTrustAnyone        bool   `yaml:"is-trust-anyone"`
	EnableOIDCCheck      bool   `yaml:"enable-oidc-check"`
	ResourcelockIdentity string `yaml:"resourcelock-identity"`
	WebhookPort          int32  `yaml:"webhook-port"`
	WebhookAllowOnErr    bool   `yaml:"webhook-allow-on-err"`
}

func (o *SecurityConfig) Sanitize() {}

func (o *SecurityConfig) Validate() error {
	// TODO Validate options config
	return nil
}
