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
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
)

type rollbackReq struct {
	Reason            string `json:"reason"`
	ExpectedVersionID *int64 `json:"expectedVersionId"`
}

func ListRuleVersions(cs consolectx.Context, kind coremodel.ResourceKind) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureVersioningEnabled(c, cs) {
			return
		}
		resp, err := service.ListRuleVersions(cs, service.RuleKindName{Kind: kind, Mesh: c.Query("mesh"), Name: c.Param("ruleName")})
		writeVersioningResp(c, resp, err)
	}
}

func GetRuleVersion(cs consolectx.Context, kind coremodel.ResourceKind) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureVersioningEnabled(c, cs) {
			return
		}
		id, ok := parseVersionID(c)
		if !ok {
			return
		}
		resp, err := service.GetRuleVersion(cs, service.RuleKindName{Kind: kind, Mesh: c.Query("mesh"), Name: c.Param("ruleName")}, id)
		writeVersioningResp(c, resp, err)
	}
}

func DiffRuleVersion(cs consolectx.Context, kind coremodel.ResourceKind) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureVersioningEnabled(c, cs) {
			return
		}
		id, ok := parseVersionID(c)
		if !ok {
			return
		}
		resp, err := service.DiffRuleVersion(cs, service.RuleKindName{Kind: kind, Mesh: c.Query("mesh"), Name: c.Param("ruleName")}, id, c.Query("against"))
		writeVersioningResp(c, resp, err)
	}
}

func RollbackRuleVersion(cs consolectx.Context, kind coremodel.ResourceKind) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureVersioningEnabled(c, cs) {
			return
		}
		id, ok := parseVersionID(c)
		if !ok {
			return
		}
		req := rollbackReq{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusOK, model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument, err.Error())))
			return
		}
		resp, err := service.RollbackRuleVersion(c.Request.Context(), cs, service.RuleKindName{Kind: kind, Mesh: c.Query("mesh"), Name: c.Param("ruleName")}, id, req.Reason, req.ExpectedVersionID, currentUser(c))
		writeVersioningResp(c, resp, err)
	}
}

func parseExpectedVersionID(c *gin.Context) (*int64, bool) {
	raw := strings.TrimSpace(c.Query("expectedVersionId"))
	if raw == "" {
		return nil, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument, "expectedVersionId must be an integer")))
		return nil, false
	}
	return &id, true
}

func mutationOptions(c *gin.Context) (service.RuleMutationOptions, bool) {
	expected, ok := parseExpectedVersionID(c)
	if !ok {
		return service.RuleMutationOptions{}, false
	}
	return service.RuleMutationOptions{ExpectedVersionID: expected, Author: currentUser(c)}, true
}

func parseVersionID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("versionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument, "versionId must be an integer")))
		return 0, false
	}
	return id, true
}

func currentUser(c *gin.Context) string {
	session := sessions.Default(c)
	if user, ok := session.Get("user").(string); ok && strings.TrimSpace(user) != "" {
		return user
	}
	return "system:unknown"
}

func ensureVersioningEnabled(c *gin.Context, cs consolectx.Context) bool {
	if cs.RuleVersioning() != nil && cs.Config().Versioning != nil && cs.Config().Versioning.Enabled {
		return true
	}
	c.JSON(http.StatusServiceUnavailable, &model.CommonResp{
		Code:    "FEATURE_DISABLED",
		Message: versioning.ErrFeatureDisabled.Error(),
	})
	return false
}

func writeVersioningResp(c *gin.Context, data any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, model.NewSuccessResp(data))
		return
	}
	var conflict *versioning.ConflictError
	var bizErr bizerror.Error
	switch {
	case errors.As(err, &conflict):
		c.JSON(http.StatusConflict, gin.H{
			"code":             "VERSION_CONFLICT",
			"message":          versioning.ErrVersionConflict.Error(),
			"currentVersionId": conflict.CurrentVersionID,
		})
	case errors.Is(err, versioning.ErrFeatureDisabled):
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "FEATURE_DISABLED", "message": err.Error()})
	case errors.Is(err, versioning.ErrVersionNotFound):
		c.JSON(http.StatusOK, model.NewBizErrorResp(bizerror.New(bizerror.NotFoundError, err.Error())))
	case errors.Is(err, versioning.ErrRollbackToDelete):
		c.JSON(http.StatusOK, model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument, err.Error())))
	case errors.As(err, &bizErr) && bizErr.Code() == bizerror.InvalidArgument:
		c.JSON(http.StatusBadRequest, model.NewBizErrorResp(bizErr))
	default:
		c.JSON(http.StatusOK, model.NewBizErrorResp(bizerror.New(bizerror.UnknownError, err.Error())))
	}
}
