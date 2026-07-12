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

package trace

import (
	"context"
	"fmt"
	"regexp"

	observabilitycfg "github.com/apache/dubbo-admin/pkg/config/observability"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/common"
)

var traceIDPattern = regexp.MustCompile(`^(?:[0-9a-fA-F]{16}|[0-9a-fA-F]{32})$`)

// GetTraceByID queries a configured provider using a validated Jaeger trace identifier.
func GetTraceByID(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	helper := common.NewArgsHelper(args)
	traceID := helper.GetString("traceId", "")
	if !traceIDPattern.MatchString(traceID) {
		return common.ErrorResult(fmt.Errorf("traceId must be a 16 or 32 character hexadecimal value")), nil
	}
	providerConfig, err := providerConfigFromContext(ctx, helper.GetString("provider", ""))
	if err != nil {
		return common.ErrorResult(err), nil
	}
	provider, err := newProvider(providerConfig)
	if err != nil {
		return common.ErrorResult(err), nil
	}
	requestCtx := context.Background()
	if ctx.AppContext() != nil {
		requestCtx = ctx.AppContext()
	}
	result, err := provider.GetTraceByID(requestCtx, traceID)
	if err != nil {
		return common.ErrorResult(err), nil
	}
	return common.JsonResult(result)
}

func providerConfigFromContext(ctx consolectx.Context, name string) (observabilitycfg.TraceProviderConfig, error) {
	if ctx == nil || ctx.Config().Observability == nil || ctx.Config().Observability.Tracing == nil {
		return observabilitycfg.TraceProviderConfig{}, fmt.Errorf("trace provider is not configured")
	}
	tracing := ctx.Config().Observability.Tracing
	if name == "" {
		provider, ok := tracing.Default()
		if !ok {
			return observabilitycfg.TraceProviderConfig{}, fmt.Errorf("default trace provider is not configured")
		}
		return provider, nil
	}
	provider, ok := tracing.Get(name)
	if !ok {
		return observabilitycfg.TraceProviderConfig{}, fmt.Errorf("trace provider %q is not configured", name)
	}
	return provider, nil
}

// Properties returns the MCP input schema properties for trace lookup.
func Properties() map[string]common.PropertyDef {
	return map[string]common.PropertyDef{
		"traceId":  {Type: "string", Description: "Jaeger 64-bit or 128-bit hexadecimal TraceID"},
		"provider": {Type: "string", Description: "Optional trace provider name; defaults to defaultProvider"},
	}
}
