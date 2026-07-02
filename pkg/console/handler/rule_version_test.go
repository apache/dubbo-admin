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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	appcfg "github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
)

type ruleVersionHandlerTestContext struct{}

func (ruleVersionHandlerTestContext) ResourceManager() manager.ResourceManager { return nil }
func (ruleVersionHandlerTestContext) CounterManager() counter.CounterManager   { return nil }
func (ruleVersionHandlerTestContext) Config() appcfg.AdminConfig               { return appcfg.AdminConfig{} }
func (ruleVersionHandlerTestContext) AppContext() context.Context              { return context.Background() }
func (ruleVersionHandlerTestContext) LockManager() lock.Lock                   { return nil }
func (ruleVersionHandlerTestContext) RuleVersioning() *versioning.Service      { return nil }

func testGinContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	return ctx, recorder
}

func decodeCommonResp(t *testing.T, recorder *httptest.ResponseRecorder) model.CommonResp {
	t.Helper()
	var resp model.CommonResp
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return resp
}

func TestEnsureVersioningAvailableUsesCommonRespContract(t *testing.T) {
	ctx, recorder := testGinContext()

	ok := ensureVersioningAvailable(ctx, ruleVersionHandlerTestContext{})

	require.False(t, ok)
	assert.Equal(t, http.StatusOK, recorder.Code)
	resp := decodeCommonResp(t, recorder)
	assert.Equal(t, string(bizerror.InternalError), resp.Code)
	assert.Equal(t, "rule history service is unavailable", resp.Message)
}

func TestWriteVersioningRespMapsBusinessErrorsToCommonResp(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{
			name: "version not found",
			err:  versioning.ErrVersionNotFound,
			code: string(bizerror.NotFoundError),
		},
		{
			name: "rollback to current",
			err:  versioning.ErrRollbackToCurrent,
			code: string(bizerror.InvalidArgument),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, recorder := testGinContext()

			writeVersioningResp(ctx, nil, tt.err)

			assert.Equal(t, http.StatusOK, recorder.Code)
			resp := decodeCommonResp(t, recorder)
			assert.Equal(t, tt.code, resp.Code)
			assert.Equal(t, tt.err.Error(), resp.Message)
		})
	}
}
