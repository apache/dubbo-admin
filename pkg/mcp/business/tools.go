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

package business

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
)

var toolNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)

// NewToolID returns the opaque identifier assigned by the control plane when
// a tool is created. Tool names are mutable, so callers must use this ID when
// addressing a tool in a draft.
func NewToolID() string {
	return uuid.NewString()
}

func CloneTool(tool *meshproto.MCPToolBinding) *meshproto.MCPToolBinding {
	if tool == nil {
		return nil
	}
	return proto.Clone(tool).(*meshproto.MCPToolBinding)
}

// FindToolIndex finds a draft tool by its stable ID. It deliberately does not
// fall back to the name because renaming a tool must not change its identity.
func FindToolIndex(snapshot *meshproto.MCPServerSnapshot, toolID string) int {
	if snapshot == nil || strings.TrimSpace(toolID) == "" {
		return -1
	}
	for i, tool := range snapshot.Tools {
		if tool != nil && tool.ToolId == toolID {
			return i
		}
	}
	return -1
}

func ValidateToolIdentity(snapshot *meshproto.MCPServerSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("server draft is required")
	}
	seenIDs := make(map[string]struct{}, len(snapshot.Tools))
	seenNames := make(map[string]struct{}, len(snapshot.Tools))
	for i, tool := range snapshot.Tools {
		if tool == nil {
			return fmt.Errorf("tool at index %d is nil", i)
		}
		if _, err := uuid.Parse(tool.ToolId); err != nil {
			return fmt.Errorf("tool at index %d has invalid toolId: %w", i, err)
		}
		if _, exists := seenIDs[tool.ToolId]; exists {
			return fmt.Errorf("toolId %q is duplicated", tool.ToolId)
		}
		seenIDs[tool.ToolId] = struct{}{}
		if !toolNamePattern.MatchString(tool.ToolName) {
			return fmt.Errorf("tool %q has invalid name", tool.ToolName)
		}
		if _, exists := seenNames[tool.ToolName]; exists {
			return fmt.Errorf("tool name %q is duplicated", tool.ToolName)
		}
		seenNames[tool.ToolName] = struct{}{}
		if strings.TrimSpace(tool.Description) == "" {
			return fmt.Errorf("tool %q description is required", tool.ToolName)
		}
		if tool.TimeoutEnabled && tool.TimeoutMillis <= 0 {
			return fmt.Errorf("tool %q timeoutMillis must be positive when timeout is enabled", tool.ToolName)
		}
	}
	return nil
}
