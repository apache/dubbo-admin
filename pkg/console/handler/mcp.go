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
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/console/util"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func RequireMCPMesh() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(c.Query("mesh")) == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest,
				model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument, "mesh is required")))
		}
	}
}

func ListMCPServers(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		servers, err := service.ListMCPServers(ctx, c.Query("mesh"))
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(servers))
	}
}

func CreateMCPServer(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := new(model.MCPServerCreateReq)
		if err := c.ShouldBindJSON(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		server, err := service.CreateMCPServer(ctx, req)
		if err != nil {
			writeMCPResult(c, nil, err)
			return
		}
		auditMCPMutation(c, "server.create", req.Mesh, req.Name)
		c.JSON(http.StatusOK, model.NewSuccessResp(server))
	}
}

func GetMCPServer(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		server, err := service.GetMCPServerInMesh(ctx, c.Query("mesh"), c.Param("serverId"))
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(server))
	}
}

func DeleteMCPServer(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := service.DeleteMCPServer(ctx, c.Query("mesh"), c.Param("serverId"), c.Query("expectedResourceVersion"))
		if err == nil {
			auditMCPMutation(c, "server.delete", c.Query("mesh"), c.Param("serverId"))
		}
		writeMCPResult(c, true, err)
	}
}

func ReplaceMCPDraft(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := new(model.MCPDraftReq)
		if err := c.ShouldBindJSON(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		server, err := service.ReplaceMCPDraft(ctx, c.Query("mesh"), c.Param("serverId"), req)
		if err == nil {
			auditMCPMutation(c, "draft.replace", c.Query("mesh"), c.Param("serverId"))
		}
		writeMCPResult(c, server, err)
	}
}

func ValidateMCPServer(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := service.ValidateMCPServer(ctx, c.Query("mesh"), c.Param("serverId"))
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(result))
	}
}

func CreateMCPTool(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := new(model.MCPToolReq)
		if err := c.ShouldBindJSON(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		server, tool, err := service.CreateMCPTool(ctx, c.Query("mesh"), c.Param("serverId"), req)
		if err != nil {
			writeMCPResult(c, nil, err)
			return
		}
		auditMCPMutation(c, "tool.create", c.Query("mesh"), c.Param("serverId"))
		c.JSON(http.StatusOK, model.NewSuccessResp(map[string]any{"server": server, "tool": tool}))
	}
}

func ListMCPTools(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		server, err := service.GetMCPServerInMesh(ctx, c.Query("mesh"), c.Param("serverId"))
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		if server.Spec == nil || server.Spec.Draft == nil {
			c.JSON(http.StatusOK, model.NewSuccessResp([]any{}))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(server.Spec.Draft.Tools))
	}
}

func GetMCPTool(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		tool, err := service.GetMCPTool(ctx, c.Query("mesh"), c.Param("serverId"), c.Param("toolId"))
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(tool))
	}
}

func ValidateMCPTool(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := service.ValidateMCPTool(ctx, c.Query("mesh"), c.Param("serverId"), c.Param("toolId"))
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(result))
	}
}

func ListMCPCredentials(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		credentials, err := service.ListMCPCredentials(ctx, c.Query("mesh"), c.Param("serverId"))
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(credentials))
	}
}

func CreateMCPCredential(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := new(model.MCPCredentialCreateReq)
		if err := c.ShouldBindJSON(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		credential, err := service.CreateMCPCredential(ctx, c.Query("mesh"), c.Param("serverId"), req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		auditMCPMutation(c, "credential.create", c.Query("mesh"), c.Param("serverId"))
		c.JSON(http.StatusOK, model.NewSuccessResp(credential))
	}
}

func RevokeMCPCredential(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		credential, err := service.RevokeMCPCredential(ctx, c.Query("mesh"), c.Param("serverId"), c.Param("credentialId"), c.Query("expectedResourceVersion"))
		if err == nil {
			auditMCPMutation(c, "credential.revoke", c.Query("mesh"), c.Param("serverId"))
		}
		writeMCPResult(c, credential, err)
	}
}

func UpdateMCPTool(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := new(model.MCPToolReq)
		if err := c.ShouldBindJSON(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		server, tool, err := service.UpdateMCPTool(ctx, c.Query("mesh"), c.Param("serverId"), c.Param("toolId"), req)
		if err != nil {
			writeMCPResult(c, nil, err)
			return
		}
		auditMCPMutation(c, "tool.update", c.Query("mesh"), c.Param("serverId"))
		c.JSON(http.StatusOK, model.NewSuccessResp(map[string]any{"server": server, "tool": tool}))
	}
}

func DeleteMCPTool(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		server, err := service.DeleteMCPTool(ctx, c.Query("mesh"), c.Param("serverId"), c.Param("toolId"), c.Query("expectedResourceVersion"))
		if err == nil {
			auditMCPMutation(c, "tool.delete", c.Query("mesh"), c.Param("serverId"))
		}
		writeMCPResult(c, server, err)
	}
}

func PublishMCPServer(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := new(model.MCPPublishReq)
		if err := c.ShouldBindJSON(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		server, err := service.PublishMCPServer(ctx, c.Query("mesh"), c.Param("serverId"), req)
		if err == nil {
			auditMCPMutation(c, "server.publish", c.Query("mesh"), c.Param("serverId"))
		}
		writeMCPResult(c, server, err)
	}
}

func writeMCPResult(c *gin.Context, value any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, model.NewSuccessResp(value))
		return
	}
	if errors.Is(err, store.ErrorResourceConflict("", "", "")) {
		c.JSON(http.StatusConflict, model.NewErrorResp(err.Error()))
		return
	}
	util.HandleServiceError(c, err)
}

func auditMCPMutation(c *gin.Context, action, mesh, serverID string) {
	logger.Sugar().Infow("MCP control-plane mutation",
		"action", action,
		"principal", sessions.Default(c).Get("user"),
		"mesh", mesh,
		"server_id", serverID,
	)
}
