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
	"time"

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
	Reason string `json:"reason"`
}

const maxRuleVersionReasonLength = 1024

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
			writeVersioningInvalidArgument(c, err.Error())
			return
		}
		if !validateRuleVersionReasonLength(c, req.Reason) {
			return
		}
		resp, err := service.RollbackRuleVersion(cs, service.RuleKindName{Kind: kind, Mesh: c.Query("mesh"), Name: c.Param("ruleName")}, id, req.Reason, currentUser(c))
		writeVersioningResp(c, resp, err)
	}
}

func validateRuleVersionReasonLength(c *gin.Context, reason string) bool {
	if len(strings.TrimSpace(reason)) <= maxRuleVersionReasonLength {
		return true
	}
	writeVersioningResp(c, nil, bizerror.New(bizerror.InvalidArgument, "reason must be at most 1024 characters"))
	return false
}

func mutationOptions(c *gin.Context) service.RuleMutationOptions {
	return service.RuleMutationOptions{Author: currentUser(c)}
}

func parseVersionID(c *gin.Context) (int64, bool) {
	id, err := parseProtocolInt64(c.Param("versionId"), false)
	if err != nil {
		writeVersioningInvalidArgument(c, "versionId must be a positive decimal string")
		return 0, false
	}
	return id, true
}

func parseProtocolInt64(raw string, allowZero bool) (int64, error) {
	if raw == "" {
		return 0, fmt.Errorf("empty id")
	}
	for i := range raw {
		if raw[i] < '0' || raw[i] > '9' {
			return 0, fmt.Errorf("invalid decimal id")
		}
	}
	if len(raw) > 1 && raw[0] == '0' {
		return 0, fmt.Errorf("invalid leading zero")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	if id == 0 && allowZero {
		return id, nil
	}
	if id <= 0 {
		return 0, fmt.Errorf("id must be positive")
	}
	return id, nil
}

func currentUser(c *gin.Context) string {
	session := sessions.Default(c)
	if user, ok := session.Get("user").(string); ok && strings.TrimSpace(user) != "" {
		return user
	}
	return "system:unknown"
}

func ensureVersioningEnabled(c *gin.Context, cs consolectx.Context) bool {
	if cs.RuleVersioning() != nil {
		return true
	}
	c.JSON(http.StatusServiceUnavailable, &model.CommonResp{
		Code:    "RULE_HISTORY_UNAVAILABLE",
		Message: "rule history service is unavailable",
	})
	return false
}

func writeVersioningResp(c *gin.Context, data any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, model.NewSuccessResp(versioningAPIData(data)))
		return
	}
	var bizErr bizerror.Error
	switch {
	case errors.Is(err, versioning.ErrVersionNotFound):
		c.JSON(http.StatusNotFound, model.NewBizErrorResp(bizerror.New(bizerror.NotFoundError, err.Error())))
	case errors.Is(err, versioning.ErrRollbackToDelete), errors.Is(err, versioning.ErrRollbackToCurrent):
		c.JSON(http.StatusBadRequest, model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument, err.Error())))
	case errors.As(err, &bizErr) && bizErr.Code() == bizerror.InvalidArgument:
		c.JSON(http.StatusBadRequest, model.NewBizErrorResp(bizErr))
	case errors.As(err, &bizErr) && bizErr.Code() == bizerror.NotFoundError:
		c.JSON(http.StatusNotFound, model.NewBizErrorResp(bizErr))
	default:
		c.JSON(http.StatusInternalServerError, model.NewBizErrorResp(bizerror.New(bizerror.UnknownError, err.Error())))
	}
}

type ruleVersionAPI struct {
	ID               string                 `json:"id"`
	RuleKind         coremodel.ResourceKind `json:"ruleKind"`
	Mesh             string                 `json:"mesh"`
	ResourceKey      string                 `json:"resourceKey"`
	RuleName         string                 `json:"ruleName"`
	VersionNo        int64                  `json:"versionNo"`
	ContentHash      string                 `json:"contentHash"`
	SpecJSON         string                 `json:"specJson"`
	Source           versioning.Source      `json:"source"`
	Operation        versioning.Operation   `json:"operation"`
	Author           string                 `json:"author"`
	Reason           string                 `json:"reason,omitempty"`
	RolledBackFromID *string                `json:"rolledBackFromId,omitempty"`
	CreatedAt        time.Time              `json:"createdAt"`
	RecordedAt       time.Time              `json:"recordedAt"`
	IsCurrent        bool                   `json:"isCurrent"`
}

type ruleVersionListAPI struct {
	Items            []ruleVersionAPI `json:"items"`
	Total            int64            `json:"total"`
	CurrentVersionID *string          `json:"currentVersionId,omitempty"`
	CurrentVersionNo int64            `json:"currentVersionNo,omitempty"`
	Deleted          bool             `json:"deleted"`
}

type ruleVersionDiffAPI struct {
	Left  ruleVersionDiffSideAPI `json:"left"`
	Right ruleVersionDiffSideAPI `json:"right"`
}

type ruleVersionDiffSideAPI struct {
	ID        string `json:"id"`
	VersionNo int64  `json:"versionNo"`
	SpecJSON  string `json:"specJson"`
}

type rollbackRuleVersionAPI struct {
	RolledBackFromID string `json:"rolledBackFromId"`
	VersionID        string `json:"versionId"`
	VersionNo        int64  `json:"versionNo"`
	Source           string `json:"source"`
	HistoryRecorded  bool   `json:"historyRecorded"`
}

func versioningAPIData(data any) any {
	switch v := data.(type) {
	case *versioning.ListResult:
		if v == nil {
			return nil
		}
		items := make([]ruleVersionAPI, 0, len(v.Items))
		for i := range v.Items {
			items = append(items, toRuleVersionAPI(&v.Items[i]))
		}
		return &ruleVersionListAPI{
			Items:            items,
			Total:            v.Total,
			CurrentVersionID: formatOptionalInt64(v.CurrentVersionID),
			CurrentVersionNo: v.CurrentVersionNo,
			Deleted:          v.Deleted,
		}
	case *versioning.Version:
		if v == nil {
			return nil
		}
		return toRuleVersionAPI(v)
	case *versioning.DiffResult:
		if v == nil {
			return nil
		}
		return &ruleVersionDiffAPI{
			Left:  toRuleVersionDiffSideAPI(v.Left),
			Right: toRuleVersionDiffSideAPI(v.Right),
		}
	case *service.RollbackResult:
		if v == nil {
			return nil
		}
		return &rollbackRuleVersionAPI{
			RolledBackFromID: formatInt64(v.RolledBackFromID),
			VersionID:        formatInt64(v.VersionID),
			VersionNo:        v.VersionNo,
			Source:           v.Source,
			HistoryRecorded:  v.HistoryRecorded,
		}
	default:
		return data
	}
}

func toRuleVersionAPI(v *versioning.Version) ruleVersionAPI {
	return ruleVersionAPI{
		ID:               formatInt64(v.ID),
		RuleKind:         v.RuleKind,
		Mesh:             v.Mesh,
		ResourceKey:      v.ResourceKey,
		RuleName:         v.RuleName,
		VersionNo:        v.VersionNo,
		ContentHash:      v.ContentHash,
		SpecJSON:         v.SpecJSON,
		Source:           v.Source,
		Operation:        v.Operation,
		Author:           v.Author,
		Reason:           v.Reason,
		RolledBackFromID: formatOptionalInt64(v.RolledBackFromID),
		CreatedAt:        v.CreatedAt,
		RecordedAt:       v.RecordedAt,
		IsCurrent:        v.IsCurrent,
	}
}

func toRuleVersionDiffSideAPI(side versioning.DiffSide) ruleVersionDiffSideAPI {
	return ruleVersionDiffSideAPI{
		ID:        formatInt64(side.ID),
		VersionNo: side.VersionNo,
		SpecJSON:  side.SpecJSON,
	}
}

func formatInt64(id int64) string {
	return strconv.FormatInt(id, 10)
}

func formatOptionalInt64(id *int64) *string {
	if id == nil {
		return nil
	}
	value := formatInt64(*id)
	return &value
}

func writeVersioningInvalidArgument(c *gin.Context, message string) {
	writeVersioningResp(c, nil, bizerror.New(bizerror.InvalidArgument, message))
}
