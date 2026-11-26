/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with this
 * work for additional information regarding copyright ownership.
 * The ASF licenses this file under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with it.
 * You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by law or agreed in writing, software is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
 */
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MatchContextValidator interface {
	ValidateMatchContext(context.Context, security.AccessContext, store.MatchContext) (store.MatchContext, error)
}
type MemoryProject struct {
	ID       string   `yaml:"id" json:"id"`
	TenantID string   `yaml:"tenant_id" json:"tenant_id"`
	OwnerIDs []string `yaml:"owner_ids" json:"owner_ids"`
}

// ProjectCatalog is a server-owned MVP catalog. Membership is not resource or
// tool authorization. An empty catalog rejects every nonempty project reference.
type ProjectCatalog []MemoryProject

func (catalog ProjectCatalog) ValidateMatchContext(ctx context.Context, a security.AccessContext, m store.MatchContext) (store.MatchContext, error) {
	if err := ctx.Err(); err != nil {
		return m, err
	}
	if err := a.ValidateBusiness(); err != nil {
		return m, err
	}
	m, err := store.NormalizeMatchContext(m)
	if err != nil {
		return m, err
	}
	if m.ProjectRef == "" {
		return m, nil
	}
	for _, p := range catalog {
		if p.ID == m.ProjectRef && p.TenantID == a.Subject.TenantID {
			for _, owner := range p.OwnerIDs {
				if owner == a.Subject.UserID {
					return m, nil
				}
			}
		}
	}
	return m, security.ErrResourceNotAccessible
}
func (r *Router) SetMatchContextValidator(v MatchContextValidator) { r.handler.matchValidator = v }
func (r *Router) SetMemoryAuditEnabled(enabled bool)               { r.handler.memoryAuditEnabled = enabled }

func (h *AgentHandler) auditMemory(c *gin.Context, a security.AccessContext, event string) {
	if !h.memoryAuditEnabled {
		return
	}
	if sink, ok := h.memoryStore.(store.UsageAuditStore); ok {
		// Match the existing best-effort audit hook: no Memory values, project
		// background, request bodies, or secrets are copied into audit records.
		_ = sink.RecordAudit(c.Request.Context(), store.AuditEvent{ID: uuid.NewString(), SchemaVersion: "v1", TenantID: a.Subject.TenantID, UserID: a.Subject.UserID, EventType: event, Status: "ok", PolicyVersion: "explicit-memory-v1", ComponentVersion: "v1", CreatedAt: time.Now()})
	}
}

func strictJSON(c *gin.Context, value any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<10)
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return store.ErrMemoryInvalid
	}
	return nil
}
func (h *AgentHandler) validateMatch(c *gin.Context, a security.AccessContext, m store.MatchContext) (store.MatchContext, error) {
	v := h.matchValidator
	if v == nil {
		v = ProjectCatalog{}
	}
	return v.ValidateMatchContext(c.Request.Context(), a, m)
}
func (h *AgentHandler) memoryAccess(c *gin.Context) (security.AccessContext, bool) {
	a, err := security.AccessContextFromContext(c.Request.Context())
	if err != nil || a.ValidateBusiness() != nil {
		c.JSON(http.StatusUnauthorized, NewAPIErrorResponse("AUTHENTICATION_REQUIRED", "Authentication required"))
		return a, false
	}
	if h.memoryStore == nil {
		c.JSON(http.StatusServiceUnavailable, NewAPIErrorResponse("MEMORY_UNAVAILABLE", "Memory storage unavailable"))
		return a, false
	}
	return a, true
}
func memoryError(c *gin.Context, err error) {
	status, code := http.StatusInternalServerError, "INTERNAL_ERROR"
	switch {
	case errors.Is(err, store.ErrMemoryInvalid):
		status, code = http.StatusBadRequest, "MEMORY_INVALID"
	case errors.Is(err, store.ErrMemoryConflict):
		status, code = http.StatusConflict, "MEMORY_VERSION_CONFLICT"
	case errors.Is(err, store.ErrMemoryNotFound):
		status, code = http.StatusNotFound, "MEMORY_NOT_FOUND"
	case errors.Is(err, store.ErrMemoryDisabled):
		status, code = http.StatusConflict, "MEMORY_DISABLED"
	case errors.Is(err, security.ErrResourceNotAccessible), errors.Is(err, store.ErrSessionNotFound), errors.Is(err, store.ErrSessionExpired):
		status, code = http.StatusNotFound, "RESOURCE_NOT_ACCESSIBLE"
	case errors.Is(err, store.ErrResourceExhausted):
		status, code = http.StatusTooManyRequests, "RESOURCE_EXHAUSTED"
	}
	c.JSON(status, NewAPIErrorResponse(code, http.StatusText(status)))
}
func (h *AgentHandler) GetMemorySettings(c *gin.Context) {
	a, ok := h.memoryAccess(c)
	if !ok {
		return
	}
	v, err := h.memoryStore.GetMemorySettings(c.Request.Context(), a)
	if err != nil {
		memoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, NewSuccessResponse(v))
}
func (h *AgentHandler) ListMemories(c *gin.Context) {
	a, ok := h.memoryAccess(c)
	if !ok {
		return
	}
	limit := 32
	if raw := c.Query("limit"); raw != "" {
		var err error
		limit, err = strconv.Atoi(raw)
		if err != nil {
			memoryError(c, store.ErrMemoryInvalid)
			return
		}
	}
	v, err := h.memoryStore.ListMemories(c.Request.Context(), a, c.Query("cursor"), limit)
	if err != nil {
		memoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, NewSuccessResponse(v))
}
func (h *AgentHandler) SaveMemory(c *gin.Context) {
	a, ok := h.memoryAccess(c)
	if !ok {
		return
	}
	var w store.MemoryWrite
	if err := strictJSON(c, &w); err != nil {
		memoryError(c, store.ErrMemoryInvalid)
		return
	}
	w.ID = c.Param("memoryId")
	var err error
	w.Applicability, err = h.validateMatch(c, a, w.Applicability)
	if err != nil {
		memoryError(c, err)
		return
	}
	v, err := h.memoryStore.SaveMemory(c.Request.Context(), a, w)
	if err != nil {
		memoryError(c, err)
		return
	}
	action := "memory.create"
	if w.ID != "" {
		action = "memory.update"
	}
	h.auditMemory(c, a, action)
	c.JSON(http.StatusOK, NewSuccessResponse(v))
}
func (h *AgentHandler) DeleteMemory(c *gin.Context) {
	a, ok := h.memoryAccess(c)
	if !ok {
		return
	}
	var w store.MemoryMutation
	if err := strictJSON(c, &w); err != nil || w.Enabled != nil {
		memoryError(c, store.ErrMemoryInvalid)
		return
	}
	if err := h.memoryStore.DeleteMemory(c.Request.Context(), a, c.Param("memoryId"), w); err != nil {
		memoryError(c, err)
		return
	}
	h.auditMemory(c, a, "memory.delete")
	c.JSON(http.StatusOK, NewSuccessResponse(nil))
}
func (h *AgentHandler) ChangeMemorySettings(c *gin.Context) { h.changeMemory(c, false) }
func (h *AgentHandler) ClearMemories(c *gin.Context)        { h.changeMemory(c, true) }
func (h *AgentHandler) changeMemory(c *gin.Context, clear bool) {
	a, ok := h.memoryAccess(c)
	if !ok {
		return
	}
	var w store.MemoryMutation
	if err := strictJSON(c, &w); err != nil || w.Version != 0 || clear && w.Enabled != nil || !clear && w.Enabled == nil {
		memoryError(c, store.ErrMemoryInvalid)
		return
	}
	v, err := h.memoryStore.ChangeMemorySettings(c.Request.Context(), a, w, clear)
	if err != nil {
		memoryError(c, err)
		return
	}
	action := "memory.enable"
	if clear {
		action = "memory.clear"
	} else if !v.Enabled {
		action = "memory.disable"
	}
	h.auditMemory(c, a, action)
	c.JSON(http.StatusOK, NewSuccessResponse(v))
}
