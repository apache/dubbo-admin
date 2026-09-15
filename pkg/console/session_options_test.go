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

package console

import (
	"net/http"
	"testing"

	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
)

func TestAdminSessionOptions(t *testing.T) {
	cfg := &configauth.Config{ExpirationTime: 3600, SessionCookieSecure: true}
	got := adminSessionOptions(cfg)
	if !got.HttpOnly || !got.Secure || got.SameSite != http.SameSiteLaxMode || got.Path != "/" || got.MaxAge != 3600 {
		t.Fatalf("adminSessionOptions() = %+v", got)
	}
}
