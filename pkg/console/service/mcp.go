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

package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
	"github.com/apache/dubbo-admin/pkg/mcp/availability"
	"github.com/apache/dubbo-admin/pkg/mcp/business"
)

func ListMCPServers(ctx consolectx.Context, mesh string) ([]*meshresource.MCPServerResource, error) {
	if err := requireMCPMesh(mesh); err != nil {
		return nil, err
	}
	return manager.ListByIndexes[*meshresource.MCPServerResource](
		ctx.ResourceManager(), meshresource.MCPServerKind,
		[]index.IndexCondition{{IndexName: index.ByMeshIndex, Value: mesh, Operator: index.Equals}},
	)
}

func GetMCPServerInMesh(ctx consolectx.Context, mesh, serverID string) (*meshresource.MCPServerResource, error) {
	if err := requireMCPMesh(mesh); err != nil {
		return nil, err
	}
	server, exists, err := manager.GetByKey[*meshresource.MCPServerResource](
		ctx.ResourceManager(), meshresource.MCPServerKind, coremodel.BuildResourceKey(mesh, serverID))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, bizerror.New(bizerror.NotFoundError, fmt.Sprintf("MCP server %q not found", serverID))
	}
	return server, nil
}

func DeleteMCPServer(ctx consolectx.Context, mesh, serverID, expectedVersion string) error {
	if strings.TrimSpace(expectedVersion) == "" {
		return bizerror.New(bizerror.InvalidArgument, "expectedResourceVersion is required")
	}
	if _, err := GetMCPServerInMesh(ctx, mesh, serverID); err != nil {
		return err
	}
	cascading, ok := ctx.ResourceManager().(manager.CascadingResourceManager)
	if !ok {
		return fmt.Errorf("resource manager does not support cascading deletion")
	}
	server := meshresource.NewMCPServerResourceWithAttributes(serverID, mesh)
	return cascading.CompareAndDeleteWithDependents(server, expectedVersion, meshresource.MCPCredentialKind, []index.IndexCondition{
		{IndexName: index.ByMeshIndex, Value: mesh, Operator: index.Equals},
		{IndexName: index.ByMCPCredentialServerID, Value: serverID, Operator: index.Equals},
	})
}

func CreateMCPServer(ctx consolectx.Context, req *model.MCPServerCreateReq) (*meshresource.MCPServerResource, error) {
	if req == nil || strings.TrimSpace(req.Mesh) == "" || strings.TrimSpace(req.Name) == "" {
		return nil, bizerror.New(bizerror.InvalidArgument, "mesh and name are required")
	}
	draft := req.Draft
	if draft == nil {
		draft = &meshproto.MCPServerSnapshot{}
	}
	draft = proto.Clone(draft).(*meshproto.MCPServerSnapshot)
	for _, tool := range draft.Tools {
		if tool != nil && strings.TrimSpace(tool.ToolId) == "" {
			tool.ToolId = business.NewToolID()
		}
	}
	prepared, err := prepareMCPDraft(ctx, req.Mesh, draft)
	if err != nil {
		return nil, err
	}
	server := meshresource.NewMCPServerResourceWithAttributes(req.Name, req.Mesh)
	server.Spec.Draft = prepared
	if err := ctx.ResourceManager().Add(server); err != nil {
		return nil, err
	}
	return server, nil
}

func ReplaceMCPDraft(ctx consolectx.Context, mesh, serverID string, req *model.MCPDraftReq) (*meshresource.MCPServerResource, error) {
	if req == nil || req.Draft == nil || strings.TrimSpace(req.ExpectedResourceVersion) == "" {
		return nil, bizerror.New(bizerror.InvalidArgument, "draft and expectedResourceVersion are required")
	}
	prepared, err := prepareMCPDraft(ctx, mesh, req.Draft)
	if err != nil {
		return nil, err
	}
	server, err := GetMCPServerInMesh(ctx, mesh, serverID)
	if err != nil {
		return nil, err
	}
	updated := server.DeepCopyObject().(*meshresource.MCPServerResource)
	updated.Spec.Draft = prepared
	if err := compareAndSwapMCPServer(ctx, updated, req.ExpectedResourceVersion); err != nil {
		return nil, err
	}
	return updated, nil
}

func ValidateMCPServer(ctx consolectx.Context, mesh, serverID string) (*model.MCPValidationResult, error) {
	server, err := GetMCPServerInMesh(ctx, mesh, serverID)
	if err != nil {
		return nil, err
	}
	if server.Spec == nil || server.Spec.Draft == nil {
		return invalidMCPValidationResult("InvalidArgument", "server draft is required", ""), nil
	}
	prepared, err := prepareMCPDraft(ctx, mesh, server.Spec.Draft)
	if err != nil {
		return mcpValidationResult(err, "")
	}
	errors, warnings, err := validateMCPProtocolEvidence(ctx, mesh, prepared)
	if err != nil {
		return nil, err
	}
	return mcpValidationIssues(errors, warnings), nil
}

func CreateMCPTool(ctx consolectx.Context, mesh, serverID string, req *model.MCPToolReq) (*meshresource.MCPServerResource, *meshproto.MCPToolBinding, error) {
	if req == nil || req.Tool == nil || strings.TrimSpace(req.ExpectedResourceVersion) == "" {
		return nil, nil, bizerror.New(bizerror.InvalidArgument, "tool and expectedResourceVersion are required")
	}
	server, err := GetMCPServerInMesh(ctx, mesh, serverID)
	if err != nil {
		return nil, nil, err
	}
	if server.Spec == nil || server.Spec.Draft == nil {
		return nil, nil, bizerror.New(bizerror.InvalidArgument, "server draft is required")
	}
	tool := proto.Clone(req.Tool).(*meshproto.MCPToolBinding)
	tool.ToolId = business.NewToolID()
	tool.Parameters = nil
	if err := resolveMCPTool(ctx, mesh, tool); err != nil {
		return nil, nil, err
	}
	draft := proto.Clone(server.Spec.Draft).(*meshproto.MCPServerSnapshot)
	draft.Tools = append(draft.Tools, tool)
	if err := business.ValidateToolIdentity(draft); err != nil {
		return nil, nil, bizerror.New(bizerror.InvalidArgument, err.Error())
	}
	updated := server.DeepCopyObject().(*meshresource.MCPServerResource)
	updated.Spec.Draft = draft
	if err := compareAndSwapMCPServer(ctx, updated, req.ExpectedResourceVersion); err != nil {
		return nil, nil, err
	}
	return updated, tool, nil
}

func UpdateMCPTool(ctx consolectx.Context, mesh, serverID, toolID string, req *model.MCPToolReq) (*meshresource.MCPServerResource, *meshproto.MCPToolBinding, error) {
	if req == nil || req.Tool == nil || strings.TrimSpace(req.ExpectedResourceVersion) == "" {
		return nil, nil, bizerror.New(bizerror.InvalidArgument, "tool and expectedResourceVersion are required")
	}
	server, err := GetMCPServerInMesh(ctx, mesh, serverID)
	if err != nil {
		return nil, nil, err
	}
	if server.Spec == nil || server.Spec.Draft == nil {
		return nil, nil, bizerror.New(bizerror.InvalidArgument, "server draft is required")
	}
	index := business.FindToolIndex(server.Spec.Draft, toolID)
	if index < 0 {
		return nil, nil, bizerror.New(bizerror.NotFoundError, fmt.Sprintf("MCP tool %q not found", toolID))
	}
	tool := proto.Clone(req.Tool).(*meshproto.MCPToolBinding)
	tool.ToolId = toolID
	if !proto.Equal(server.Spec.Draft.Tools[index].Operation, tool.Operation) {
		tool.Parameters = nil
	}
	if err := resolveMCPTool(ctx, mesh, tool); err != nil {
		return nil, nil, err
	}
	draft := proto.Clone(server.Spec.Draft).(*meshproto.MCPServerSnapshot)
	draft.Tools[index] = tool
	if err := business.ValidateToolIdentity(draft); err != nil {
		return nil, nil, bizerror.New(bizerror.InvalidArgument, err.Error())
	}
	updated := server.DeepCopyObject().(*meshresource.MCPServerResource)
	updated.Spec.Draft = draft
	if err := compareAndSwapMCPServer(ctx, updated, req.ExpectedResourceVersion); err != nil {
		return nil, nil, err
	}
	return updated, tool, nil
}

func DeleteMCPTool(ctx consolectx.Context, mesh, serverID, toolID, expectedVersion string) (*meshresource.MCPServerResource, error) {
	if strings.TrimSpace(expectedVersion) == "" {
		return nil, bizerror.New(bizerror.InvalidArgument, "expectedResourceVersion is required")
	}
	server, err := GetMCPServerInMesh(ctx, mesh, serverID)
	if err != nil {
		return nil, err
	}
	if server.Spec == nil || server.Spec.Draft == nil {
		return nil, bizerror.New(bizerror.InvalidArgument, "server draft is required")
	}
	toolIndex := business.FindToolIndex(server.Spec.Draft, toolID)
	if toolIndex < 0 {
		return nil, bizerror.New(bizerror.NotFoundError, fmt.Sprintf("MCP tool %q not found", toolID))
	}
	draft := proto.Clone(server.Spec.Draft).(*meshproto.MCPServerSnapshot)
	draft.Tools = append(draft.Tools[:toolIndex], draft.Tools[toolIndex+1:]...)
	updated := server.DeepCopyObject().(*meshresource.MCPServerResource)
	updated.Spec.Draft = draft
	if err := compareAndSwapMCPServer(ctx, updated, expectedVersion); err != nil {
		return nil, err
	}
	return updated, nil
}

func PublishMCPServer(ctx consolectx.Context, mesh, serverID string, req *model.MCPPublishReq) (*meshresource.MCPServerResource, error) {
	if req == nil || strings.TrimSpace(req.ExpectedResourceVersion) == "" {
		return nil, bizerror.New(bizerror.InvalidArgument, "expectedResourceVersion is required")
	}
	server, err := GetMCPServerInMesh(ctx, mesh, serverID)
	if err != nil {
		return nil, err
	}
	if server.Spec == nil || server.Spec.Draft == nil {
		return nil, bizerror.New(bizerror.InvalidArgument, "server draft is required")
	}
	prepared, err := prepareMCPDraft(ctx, mesh, server.Spec.Draft)
	if err != nil {
		return nil, err
	}
	errors, _, err := validateMCPProtocolEvidence(ctx, mesh, prepared)
	if err != nil {
		return nil, err
	}
	if len(errors) > 0 {
		return nil, bizerror.New(bizerror.InvalidArgument, errors[0].Message)
	}
	updated := server.DeepCopyObject().(*meshresource.MCPServerResource)
	if updated.Spec.Published == nil {
		updated.Spec.Published = &meshproto.MCPPublishedSnapshot{}
	}
	updated.Spec.Published.Revision++
	updated.Spec.Published.PublishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	updated.Spec.Published.Snapshot = prepared
	if err := compareAndSwapMCPServer(ctx, updated, req.ExpectedResourceVersion); err != nil {
		return nil, err
	}
	return updated, nil
}

func validateMCPProtocolEvidence(ctx consolectx.Context, mesh string, snapshot *meshproto.MCPServerSnapshot) ([]model.MCPValidationIssue, []model.MCPValidationIssue, error) {
	validationErrors := make([]model.MCPValidationIssue, 0)
	warnings := make([]model.MCPValidationIssue, 0)
	metadata, err := manager.ListByIndexes[*meshresource.ServiceProviderMetadataResource](
		ctx.ResourceManager(), meshresource.ServiceProviderMetadataKind,
		[]index.IndexCondition{{IndexName: index.ByMeshIndex, Value: mesh, Operator: index.Equals}},
	)
	if err != nil {
		return nil, nil, err
	}
	instances, err := manager.ListByIndexes[*meshresource.RPCInstanceResource](
		ctx.ResourceManager(), meshresource.RPCInstanceKind,
		[]index.IndexCondition{{IndexName: index.ByMeshIndex, Value: mesh, Operator: index.Equals}},
	)
	if err != nil {
		return nil, nil, err
	}
	evidence := availability.Snapshot{
		Providers: metadata,
		Instances: instances,
		Sync:      availability.SyncState{InitialSyncComplete: true, Healthy: true},
	}
	for _, tool := range snapshot.Tools {
		if tool == nil || tool.Operation == nil {
			continue
		}
		identity := availability.Identity{
			Mesh: mesh, ServiceName: tool.Operation.ServiceName,
			Group: tool.Operation.Group, Version: tool.Operation.Version,
		}
		result := availability.Evaluate(evidence, identity, tool.Operation.Protocol)
		if availability.ConfirmedProtocolIncompatibility(result) {
			validationErrors = append(validationErrors, model.MCPValidationIssue{
				Code: "ProtocolIncompatible",
				Message: fmt.Sprintf("service %s has available providers, but none support protocol %q",
					tool.Operation.ServiceName, tool.Operation.Protocol),
				ToolID: tool.ToolId,
			})
			continue
		}
		if !result.Evidence.Available {
			warnings = append(warnings, model.MCPValidationIssue{
				Code: "NoAvailableProvider",
				Message: fmt.Sprintf("service %s has no available provider; protocol compatibility cannot be confirmed",
					tool.Operation.ServiceName),
				ToolID: tool.ToolId,
			})
			continue
		}
		if !result.Evidence.Complete {
			warnings = append(warnings, model.MCPValidationIssue{
				Code: "ProtocolEvidenceIncomplete",
				Message: fmt.Sprintf("service %s has incomplete protocol evidence; compatibility cannot be confirmed",
					tool.Operation.ServiceName),
				ToolID: tool.ToolId,
			})
		}
	}
	return validationErrors, warnings, nil
}

func compareAndSwapMCPServer(ctx consolectx.Context, updated *meshresource.MCPServerResource, expectedVersion string) error {
	return compareAndSwapResource(ctx, updated, expectedVersion)
}

func compareAndSwapResource(ctx consolectx.Context, updated coremodel.Resource, expectedVersion string) error {
	conditional, ok := ctx.ResourceManager().(manager.ConditionalResourceManager)
	if !ok {
		return fmt.Errorf("resource manager does not support conditional mutations")
	}
	return conditional.CompareAndSwap(updated, expectedVersion)
}

func resolveMCPTool(ctx consolectx.Context, mesh string, tool *meshproto.MCPToolBinding) error {
	if tool == nil || tool.Operation == nil {
		return bizerror.New(bizerror.InvalidArgument, "tool operation is required")
	}
	metadata, err := manager.ListByIndexes[*meshresource.ServiceProviderMetadataResource](
		ctx.ResourceManager(), meshresource.ServiceProviderMetadataKind,
		[]index.IndexCondition{{IndexName: index.ByMeshIndex, Value: mesh, Operator: index.Equals}},
	)
	if err != nil {
		return err
	}
	contract, err := business.ResolveOperation(mesh, tool.Operation, tool.Parameters, metadata)
	if err != nil {
		return bizerror.New(bizerror.InvalidArgument, err.Error())
	}
	tool.Parameters = contract.Parameters
	tool.InputSchema = contract.InputSchema
	tool.InputFingerprint = contract.InputFingerprint
	tool.OutputFingerprint = contract.OutputFingerprint
	tool.RelatedTypes = contract.RelatedTypes
	tool.ReturnType = contract.ReturnType
	return nil
}

func prepareMCPDraft(ctx consolectx.Context, mesh string, draft *meshproto.MCPServerSnapshot) (*meshproto.MCPServerSnapshot, error) {
	if draft == nil {
		return nil, bizerror.New(bizerror.InvalidArgument, "server draft is required")
	}
	if strings.TrimSpace(draft.Description) == "" {
		return nil, bizerror.New(bizerror.InvalidArgument, "server draft description is required")
	}
	prepared := proto.Clone(draft).(*meshproto.MCPServerSnapshot)
	for _, tool := range prepared.Tools {
		if tool != nil && strings.TrimSpace(tool.ToolId) == "" {
			return nil, bizerror.New(bizerror.InvalidArgument, "toolId is required for existing draft tools")
		}
		if err := resolveMCPTool(ctx, mesh, tool); err != nil {
			return nil, err
		}
	}
	if err := business.ValidateToolIdentity(prepared); err != nil {
		return nil, bizerror.New(bizerror.InvalidArgument, err.Error())
	}
	return prepared, nil
}

func GetMCPTool(ctx consolectx.Context, mesh, serverID, toolID string) (*meshproto.MCPToolBinding, error) {
	server, err := GetMCPServerInMesh(ctx, mesh, serverID)
	if err != nil {
		return nil, err
	}
	if server.Spec == nil || server.Spec.Draft == nil {
		return nil, bizerror.New(bizerror.NotFoundError, fmt.Sprintf("MCP tool %q not found", toolID))
	}
	toolIndex := business.FindToolIndex(server.Spec.Draft, toolID)
	if toolIndex < 0 {
		return nil, bizerror.New(bizerror.NotFoundError, fmt.Sprintf("MCP tool %q not found", toolID))
	}
	return business.CloneTool(server.Spec.Draft.Tools[toolIndex]), nil
}

func ValidateMCPTool(ctx consolectx.Context, mesh, serverID, toolID string) (*model.MCPValidationResult, error) {
	tool, err := GetMCPTool(ctx, mesh, serverID, toolID)
	if err != nil {
		return nil, err
	}
	if err := resolveMCPTool(ctx, mesh, tool); err != nil {
		return mcpValidationResult(err, toolID)
	}
	if err := business.ValidateToolIdentity(&meshproto.MCPServerSnapshot{Tools: []*meshproto.MCPToolBinding{tool}}); err != nil {
		return mcpValidationResult(bizerror.New(bizerror.InvalidArgument, err.Error()), toolID)
	}
	validationErrors, warnings, err := validateMCPProtocolEvidence(
		ctx, mesh, &meshproto.MCPServerSnapshot{Tools: []*meshproto.MCPToolBinding{tool}},
	)
	if err != nil {
		return nil, err
	}
	for i := range validationErrors {
		validationErrors[i].ToolID = toolID
	}
	for i := range warnings {
		warnings[i].ToolID = toolID
	}
	return mcpValidationIssues(validationErrors, warnings), nil
}

func mcpValidationResult(err error, toolID string) (*model.MCPValidationResult, error) {
	result := &model.MCPValidationResult{
		Valid:    err == nil,
		Errors:   make([]model.MCPValidationIssue, 0),
		Warnings: make([]model.MCPValidationIssue, 0),
	}
	if err == nil {
		return result, nil
	}
	var validationError bizerror.Error
	if !errors.As(err, &validationError) || validationError.Code() != bizerror.InvalidArgument {
		return nil, err
	}
	result.Errors = append(result.Errors, model.MCPValidationIssue{
		Code:    string(validationError.Code()),
		Message: validationError.Message(),
		ToolID:  toolID,
	})
	return result, nil
}

func mcpValidationIssues(errors, warnings []model.MCPValidationIssue) *model.MCPValidationResult {
	if errors == nil {
		errors = make([]model.MCPValidationIssue, 0)
	}
	if warnings == nil {
		warnings = make([]model.MCPValidationIssue, 0)
	}
	return &model.MCPValidationResult{
		Valid:    len(errors) == 0,
		Errors:   errors,
		Warnings: warnings,
	}
}

func invalidMCPValidationResult(code, message, toolID string) *model.MCPValidationResult {
	return &model.MCPValidationResult{
		Valid: false,
		Errors: []model.MCPValidationIssue{{
			Code:    code,
			Message: message,
			ToolID:  toolID,
		}},
		Warnings: make([]model.MCPValidationIssue, 0),
	}
}

func requireMCPMesh(mesh string) error {
	if strings.TrimSpace(mesh) == "" {
		return bizerror.New(bizerror.InvalidArgument, "mesh is required")
	}
	return nil
}

type MCPCredentialToken struct {
	Credential model.MCPCredentialView `json:"credential"`
	Token      string                  `json:"token"`
}

func credentialView(credential *meshresource.MCPCredentialResource) model.MCPCredentialView {
	return model.MCPCredentialView{
		ID:              credential.Name,
		Mesh:            credential.Mesh,
		ServerID:        credential.Spec.ServerId,
		ResourceVersion: credential.ResourceVersion,
		DisplayName:     credential.Spec.DisplayName,
		Status:          credential.Spec.Status,
		ExpiresAt:       credential.Spec.ExpiresAt,
		CreatedAt:       credential.Spec.CreatedAt,
	}
}

func ListMCPCredentials(ctx consolectx.Context, mesh, serverID string) ([]model.MCPCredentialView, error) {
	if _, err := GetMCPServerInMesh(ctx, mesh, serverID); err != nil {
		return nil, err
	}
	credentials, err := manager.ListByIndexes[*meshresource.MCPCredentialResource](
		ctx.ResourceManager(), meshresource.MCPCredentialKind,
		[]index.IndexCondition{
			{IndexName: index.ByMeshIndex, Value: mesh, Operator: index.Equals},
			{IndexName: index.ByMCPCredentialServerID, Value: serverID, Operator: index.Equals},
		},
	)
	if err != nil {
		return nil, err
	}
	views := make([]model.MCPCredentialView, 0, len(credentials))
	for _, credential := range credentials {
		if credential != nil && credential.Spec != nil {
			views = append(views, credentialView(credential))
		}
	}
	return views, nil
}

func CreateMCPCredential(ctx consolectx.Context, mesh, serverID string, req *model.MCPCredentialCreateReq) (*MCPCredentialToken, error) {
	if req == nil || strings.TrimSpace(req.DisplayName) == "" {
		return nil, bizerror.New(bizerror.InvalidArgument, "displayName is required")
	}
	if _, err := GetMCPServerInMesh(ctx, mesh, serverID); err != nil {
		return nil, err
	}
	var expiresAt time.Time
	if strings.TrimSpace(req.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil || !parsed.After(time.Now().UTC()) {
			return nil, bizerror.New(bizerror.InvalidArgument, "expiresAt must be a future RFC3339 timestamp when provided")
		}
		expiresAt = parsed
	}
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("generate credential secret: %w", err)
	}
	credentialID := uuid.NewString()
	secret := hex.EncodeToString(secretBytes)
	digest := sha256.Sum256([]byte(secret))
	credential := meshresource.NewMCPCredentialResourceWithAttributes(credentialID, mesh)
	credential.Spec = &meshproto.MCPCredential{
		ServerId:    serverID,
		DisplayName: req.DisplayName,
		SecretHash:  "sha256:" + hex.EncodeToString(digest[:]),
		Status:      "active",
		ExpiresAt:   formatMCPExpiration(expiresAt),
		CreatedAt:   time.Now().UTC().Format(time.RFC3339Nano),
	}
	dependent, ok := ctx.ResourceManager().(manager.DependentResourceManager)
	if !ok {
		return nil, fmt.Errorf("resource manager does not support dependent creation")
	}
	if err := dependent.AddWithParent(meshresource.NewMCPServerResourceWithAttributes(serverID, mesh), credential); err != nil {
		return nil, err
	}
	return &MCPCredentialToken{
		Credential: credentialView(credential),
		Token:      "mcp_" + credentialID + "." + secret,
	}, nil
}

func formatMCPExpiration(expiresAt time.Time) string {
	if expiresAt.IsZero() {
		return ""
	}
	return expiresAt.UTC().Format(time.RFC3339)
}

func RevokeMCPCredential(ctx consolectx.Context, mesh, serverID, credentialID, expectedVersion string) (*model.MCPCredentialView, error) {
	if strings.TrimSpace(expectedVersion) == "" {
		return nil, bizerror.New(bizerror.InvalidArgument, "expectedResourceVersion is required")
	}
	credential, exists, err := manager.GetByKey[*meshresource.MCPCredentialResource](
		ctx.ResourceManager(), meshresource.MCPCredentialKind, coremodel.BuildResourceKey(mesh, credentialID))
	if err != nil {
		return nil, err
	}
	if !exists || credential.Spec == nil || credential.Spec.ServerId != serverID {
		return nil, bizerror.New(bizerror.NotFoundError, fmt.Sprintf("MCP credential %q not found", credentialID))
	}
	updated := credential.DeepCopyObject().(*meshresource.MCPCredentialResource)
	updated.Spec.Status = "revoked"
	if err := compareAndSwapResource(ctx, updated, expectedVersion); err != nil {
		return nil, err
	}
	view := credentialView(updated)
	return &view, nil
}
