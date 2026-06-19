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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
)

func TestWriteVersioningRespHTTPStatusMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)

	currentVersionID := int64(42)
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantJSON   string
	}{
		{
			name:       "version not found",
			err:        versioning.ErrVersionNotFound,
			wantStatus: http.StatusNotFound,
			wantJSON:   `{"code":"NotFoundError","message":"rule version not found","data":null}`,
		},
		{
			name:       "rollback to delete marker",
			err:        versioning.ErrRollbackToDelete,
			wantStatus: http.StatusBadRequest,
			wantJSON:   `{"code":"InvalidArgument","message":"cannot roll back to a deleted rule version","data":null}`,
		},
		{
			name:       "biz invalid argument",
			err:        bizerror.New(bizerror.InvalidArgument, "rollback reason is required"),
			wantStatus: http.StatusBadRequest,
			wantJSON:   `{"code":"InvalidArgument","message":"rollback reason is required","data":null}`,
		},
		{
			name:       "biz not found",
			err:        bizerror.New(bizerror.NotFoundError, "rule version not found"),
			wantStatus: http.StatusNotFound,
			wantJSON:   `{"code":"NotFoundError","message":"rule version not found","data":null}`,
		},
		{
			name:       "version conflict",
			err:        &versioning.ConflictError{CurrentVersionID: &currentVersionID},
			wantStatus: http.StatusConflict,
			wantJSON:   `{"code":"VERSION_CONFLICT","message":"rule version conflict","currentVersionId":"42"}`,
		},
		{
			name:       "pending intent",
			err:        &versioning.IntentPendingError{IntentID: 99},
			wantStatus: http.StatusConflict,
			wantJSON:   `{"code":"VERSION_LEDGER_PENDING","message":"rule version intent is pending","intentId":"99"}`,
		},
		{
			name:       "feature disabled",
			err:        versioning.ErrFeatureDisabled,
			wantStatus: http.StatusServiceUnavailable,
			wantJSON:   `{"code":"FEATURE_DISABLED","message":"rule versioning is disabled"}`,
		},
		{
			name:       "unknown error",
			err:        errors.New("storage exploded"),
			wantStatus: http.StatusInternalServerError,
			wantJSON:   `{"code":"UnknownError","message":"storage exploded","data":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)

			writeVersioningResp(c, nil, tt.err)

			require.Equal(t, tt.wantStatus, recorder.Code)
			assert.JSONEq(t, tt.wantJSON, recorder.Body.String())
		})
	}
}

func TestWriteVersioningRespSuccessIncludesRollbackCommitFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	writeVersioningResp(c, &service.RollbackResult{
		RolledBackFromID: 123,
		VersionID:        456,
		VersionNo:        4,
		Source:           "ROLLBACK",
		Committed:        true,
	}, nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{
		"code":"Success",
		"message":"success",
		"data":{
			"rolledBackFromId":"123",
			"versionId":"456",
			"versionNo":4,
			"source":"ROLLBACK",
			"committed":true
		}
	}`, recorder.Body.String())
}

func TestWriteVersioningRespSerializesLargeVersionIDsAsStrings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	rolledBackFromID := int64(7473321752550968336)
	writeVersioningResp(c, &versioning.ListResult{
		Items: []versioning.Version{
			{
				ID:               7473321752550968337,
				VersionNo:        2,
				RolledBackFromID: &rolledBackFromID,
				IsCurrent:        true,
			},
		},
		Total:            1,
		CurrentVersionID: ptrInt64(7473321752550968337),
		CurrentVersionNo: 2,
	}, nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	var body struct {
		Data struct {
			Items []struct {
				ID               string `json:"id"`
				RolledBackFromID string `json:"rolledBackFromId"`
			} `json:"items"`
			CurrentVersionID string `json:"currentVersionId"`
			CurrentVersionNo int64  `json:"currentVersionNo"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body.Data.Items, 1)
	assert.Equal(t, "7473321752550968337", body.Data.Items[0].ID)
	assert.Equal(t, "7473321752550968336", body.Data.Items[0].RolledBackFromID)
	assert.Equal(t, "7473321752550968337", body.Data.CurrentVersionID)
	assert.Equal(t, int64(2), body.Data.CurrentVersionNo)
}

func TestParseJSONInt64AcceptsStringID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	id, ok := parseJSONInt64(c, json.RawMessage(`"7473321752550968337"`), "expectedVersionId")

	require.True(t, ok)
	require.NotNil(t, id)
	assert.Equal(t, int64(7473321752550968337), *id)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func ptrInt64(v int64) *int64 {
	return &v
}

func TestParseJSONInt64AcceptsDeletedStateSentinel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	id, ok := parseJSONInt64(c, json.RawMessage(`"0"`), "expectedVersionId")

	require.True(t, ok)
	require.NotNil(t, id)
	assert.Equal(t, int64(0), *id)
	assert.Equal(t, http.StatusOK, recorder.Code)
}
