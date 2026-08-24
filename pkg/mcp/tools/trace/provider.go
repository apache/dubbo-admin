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

	observabilitycfg "github.com/apache/dubbo-admin/pkg/config/observability"
)

// Provider hides backend wire formats behind a stable trace lookup contract.
type Provider interface {
	GetTraceByID(ctx context.Context, traceID string) (*Trace, error)
}

func newProvider(cfg observabilitycfg.TraceProviderConfig) (Provider, error) {
	switch cfg.Type {
	case observabilitycfg.TraceProviderJaeger:
		return newJaegerProvider(cfg)
	default:
		return nil, fmt.Errorf("unsupported trace provider type: %s", cfg.Type)
	}
}
