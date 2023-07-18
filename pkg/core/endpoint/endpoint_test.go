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

package endpoint_test

import (
	"testing"

	"github.com/apache/dubbo-admin/pkg/core/endpoint"

	"github.com/stretchr/testify/assert"
)

func TestToString(t *testing.T) {
	t.Parallel()

	endpoint := &endpoint.Endpoint{}

	assert.Equal(t, "{}", endpoint.ToString())

	endpoint.SpiffeID = "spiffe://example.com/ns/default/sa/default"

	assert.Equal(t, "{\"spiffeID\":\"spiffe://example.com/ns/default/sa/default\"}", endpoint.ToString())
}
