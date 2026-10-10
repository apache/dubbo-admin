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

package server

import "testing"

func TestAuthenticationSpecRoleDefaultsDistinguishMissingAndEmpty(t *testing.T) {
	missing := AuthenticationSpec{}
	missing.applyDefaults()
	if len(missing.Roles) != 1 || missing.Roles[0] != "user" {
		t.Fatalf("missing roles defaulted to %#v, want [user]", missing.Roles)
	}

	explicitEmpty := AuthenticationSpec{Roles: []string{}}
	explicitEmpty.applyDefaults()
	if explicitEmpty.Roles == nil || len(explicitEmpty.Roles) != 0 {
		t.Fatalf("explicit empty roles changed to %#v", explicitEmpty.Roles)
	}
}
