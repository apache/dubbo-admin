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

package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
)

func TestPostAffinityRuleRejectsInvalidRuntimeContract(t *testing.T) {
	ctx, recorder := testGinContext()
	ctx.Request = httptest.NewRequest(http.MethodPost, "/affinity-rule/demo.affinity-router", strings.NewReader(`{
		"configVersion":"v3.1", "scope":"application", "key":"demo", "enabled":true,
		"runtime":true, "affinityAware":{"key":"region", "ratio":101}
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "ruleName", Value: "demo.affinity-router"}}

	PostAffinityRuleWithRuleName(ruleVersionHandlerTestContext{})(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, string(bizerror.InvalidArgument), decodeCommonResp(t, recorder).Code)
}

func TestPostScriptRuleRejectsUnsupportedScriptType(t *testing.T) {
	ctx, recorder := testGinContext()
	ctx.Request = httptest.NewRequest(http.MethodPost, "/script-rule/demo.script-router", strings.NewReader(`{
		"configVersion":"v3.0", "scope":"application", "key":"demo", "enabled":true,
		"type":"groovy", "script":"return invokers;"
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "ruleName", Value: "demo.script-router"}}

	PostScriptRuleWithRuleName(ruleVersionHandlerTestContext{})(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, string(bizerror.InvalidArgument), decodeCommonResp(t, recorder).Code)
}

func TestRouterRuleHandlersRejectWrongRuleSuffix(t *testing.T) {
	ctx, recorder := testGinContext()
	ctx.Request = httptest.NewRequest(http.MethodPost, "/script-rule/demo.affinity-router", nil)
	ctx.Params = gin.Params{{Key: "ruleName", Value: "demo.affinity-router"}}

	PostScriptRuleWithRuleName(ruleVersionHandlerTestContext{})(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, string(bizerror.InvalidArgument), decodeCommonResp(t, recorder).Code)
}
