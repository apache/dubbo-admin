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
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/console/util"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
)

const maxRuleVersionReasonLength = 1024

func ListRuleVersions(cs consolectx.Context, kind coremodel.ResourceKind) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureVersioningAvailable(c, cs) {
			return
		}
		resp, err := service.ListRuleVersions(cs, ruleRef(c, kind))
		if writeVersioningError(c, err) {
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(toRuleVersionListResp(resp)))
	}
}

func GetRuleVersion(cs consolectx.Context, kind coremodel.ResourceKind) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureVersioningAvailable(c, cs) {
			return
		}
		versionNo, ok := parseVersionNo(c)
		if !ok {
			return
		}
		resp, err := service.GetRuleVersion(cs, ruleRef(c, kind), versionNo)
		if writeVersioningError(c, err) {
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(toRuleVersionResp(resp)))
	}
}

func DiffRuleVersion(cs consolectx.Context, kind coremodel.ResourceKind) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureVersioningAvailable(c, cs) {
			return
		}
		versionNo, ok := parseVersionNo(c)
		if !ok {
			return
		}
		resp, err := service.DiffRuleVersion(cs, ruleRef(c, kind), versionNo, c.Query("against"))
		if writeVersioningError(c, err) {
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(toRuleVersionDiffResp(resp)))
	}
}

func RollbackRuleVersion(cs consolectx.Context, kind coremodel.ResourceKind) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureVersioningAvailable(c, cs) {
			return
		}
		versionNo, ok := parseVersionNo(c)
		if !ok {
			return
		}
		req := model.RollbackRuleVersionReq{}
		if err := c.ShouldBindJSON(&req); err != nil {
			writeVersioningInvalidArgument(c, err.Error())
			return
		}
		if !validateRuleVersionReasonLength(c, req.Reason) {
			return
		}
		resp, err := service.RollbackRuleVersion(cs, ruleRef(c, kind), versionNo, req.Reason, currentUser(c))
		if writeVersioningError(c, err) {
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(toRollbackRuleVersionResp(resp)))
	}
}

func validateRuleVersionReasonLength(c *gin.Context, reason string) bool {
	if len(strings.TrimSpace(reason)) <= maxRuleVersionReasonLength {
		return true
	}
	writeVersioningInvalidArgument(c, "reason must be at most 1024 characters")
	return false
}

func mutationOptions(c *gin.Context) service.RuleMutationOptions {
	return service.RuleMutationOptions{Author: currentUser(c)}
}

func ruleRef(c *gin.Context, kind coremodel.ResourceKind) service.RuleRef {
	return service.RuleRef{Kind: kind, Mesh: c.Query("mesh"), Name: c.Param("ruleName")}
}

func parseVersionNo(c *gin.Context) (int64, bool) {
	versionNo, err := parsePositiveInt64(c.Param("versionNo"))
	if err != nil {
		writeVersioningInvalidArgument(c, "versionNo must be a positive decimal string")
		return 0, false
	}
	return versionNo, true
}

func parsePositiveInt64(raw string) (int64, error) {
	if raw == "" {
		return 0, fmt.Errorf("empty value")
	}
	for i := range raw {
		if raw[i] < '0' || raw[i] > '9' {
			return 0, fmt.Errorf("invalid decimal value")
		}
	}
	if len(raw) > 1 && raw[0] == '0' {
		return 0, fmt.Errorf("invalid leading zero")
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("value must be positive")
	}
	return value, nil
}

func currentUser(c *gin.Context) string {
	session := sessions.Default(c)
	if user, ok := session.Get("user").(string); ok && strings.TrimSpace(user) != "" {
		return user
	}
	return "system:unknown"
}

func ensureVersioningAvailable(c *gin.Context, cs consolectx.Context) bool {
	if cs.RuleVersioning() != nil {
		return true
	}
	util.HandleServiceError(c, bizerror.New(bizerror.InternalError, "rule history service is unavailable"))
	return false
}

func writeVersioningError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	util.HandleServiceError(c, versioningServiceError(err))
	return true
}

func versioningServiceError(err error) error {
	var bizErr bizerror.Error
	switch {
	case errors.Is(err, versioning.ErrVersionNotFound):
		return bizerror.New(bizerror.NotFoundError, err.Error())
	case errors.Is(err, versioning.ErrRollbackToDelete), errors.Is(err, versioning.ErrRollbackToCurrent):
		return bizerror.New(bizerror.InvalidArgument, err.Error())
	case errors.As(err, &bizErr):
		return bizErr
	default:
		return err
	}
}

func toRuleVersionResp(version *versioning.Version) model.RuleVersionResp {
	return model.RuleVersionResp{
		RuleKind:                version.RuleKind,
		Mesh:                    version.Mesh,
		ResourceKey:             version.ResourceKey,
		RuleName:                version.RuleName,
		VersionNo:               version.VersionNo,
		ContentHash:             version.ContentHash,
		SpecJSON:                version.SpecJSON,
		Source:                  version.Source,
		Operation:               version.Operation,
		Author:                  version.Author,
		Reason:                  version.Reason,
		RolledBackFromVersionNo: version.RolledBackFromVersionNo,
		CreatedAt:               version.CreatedAt,
		RecordedAt:              version.RecordedAt,
		IsLatestRecorded:        version.IsLatestRecorded,
	}
}

func toRuleVersionListResp(result *versioning.ListResult) model.RuleVersionListResp {
	items := make([]model.RuleVersionResp, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, toRuleVersionResp(&result.Items[i]))
	}
	return model.RuleVersionListResp{
		Items:                   items,
		Total:                   result.Total,
		LatestRecordedVersionNo: result.LatestRecordedVersionNo,
		LatestRecordedDeleted:   result.LatestRecordedDeleted,
	}
}

func toRuleVersionDiffResp(result *versioning.DiffResult) model.RuleVersionDiffResp {
	return model.RuleVersionDiffResp{
		Left:  toRuleVersionDiffSideResp(result.Left),
		Right: toRuleVersionDiffSideResp(result.Right),
	}
}

func toRuleVersionDiffSideResp(side versioning.DiffSide) model.RuleVersionDiffSideResp {
	return model.RuleVersionDiffSideResp{
		VersionNo: side.VersionNo,
		SpecJSON:  side.SpecJSON,
	}
}

func toRollbackRuleVersionResp(result *service.RollbackResult) model.RollbackRuleVersionResp {
	return model.RollbackRuleVersionResp{
		RolledBackFromVersionNo: result.RolledBackFromVersionNo,
		VersionNo:               result.VersionNo,
		Source:                  result.Source,
	}
}

func writeVersioningInvalidArgument(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument, message)))
}
