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

package invocation

import (
	"context"
	"errors"
	"strconv"

	"dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	clusterbase "dubbo.apache.org/dubbo-go/v3/cluster/cluster/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

const safeClusterName = "mcp-single-attempt"

var errUnsafeGovernance = errors.New("governance cannot guarantee a single RPC attempt")

func init() {
	extension.SetCluster(safeClusterName, func() cluster.Cluster { return safeCluster{} })
}

type safeCluster struct{}

func (safeCluster) Join(dir directory.Directory) base.Invoker {
	return &safeInvoker{dir: dir}
}

type safeInvoker struct {
	dir directory.Directory
}

func (s *safeInvoker) GetURL() *common.URL { return s.dir.GetURL() }
func (s *safeInvoker) IsAvailable() bool   { return s.dir.IsAvailable() }
func (s *safeInvoker) Destroy()            { s.dir.Destroy() }

func (s *safeInvoker) Invoke(ctx context.Context, invocation base.Invocation) result.Result {
	url := s.dir.GetURL()
	if url == nil || url.SubURL == nil || !safeURL(url.SubURL, invocation.ActualMethodName()) {
		return &result.RPCResult{Err: errUnsafeGovernance}
	}
	// Directory.List uses the SDK router chain. Check every candidate, because
	// its load balancer may choose any one of them after this preflight.
	candidates := s.dir.List(invocation)
	if len(candidates) == 0 {
		return &result.RPCResult{Err: errNoProvider}
	}
	for _, candidate := range candidates {
		if candidate == nil || !safeURL(candidate.GetURL(), invocation.ActualMethodName()) {
			return &result.RPCResult{Err: errUnsafeGovernance}
		}
	}
	// Use dubbo-go's load balancer directly. Cluster interceptors can wrap
	// failfast with another dispatch, so the selected Provider is invoked once.
	loadBalance := clusterbase.GetLoadBalance(candidates[0], invocation.ActualMethodName())
	selected := loadBalance.Select(candidates, invocation)
	if selected == nil || !selected.IsAvailable() {
		return &result.RPCResult{Err: errNoProvider}
	}
	found := false
	for _, candidate := range candidates {
		if candidate == selected {
			found = true
			break
		}
	}
	if !found || !safeURL(selected.GetURL(), invocation.ActualMethodName()) {
		return &result.RPCResult{Err: errUnsafeGovernance}
	}
	return selected.Invoke(ctx, invocation)
}

func safeURL(url *common.URL, method string) bool {
	if url == nil {
		return false
	}
	cluster := url.GetParam(constant.ClusterKey, "")
	if cluster != safeClusterName && cluster != constant.ClusterKeyFailfast {
		return false
	}
	methodCluster := url.GetMethodParam(method, constant.ClusterKey, "")
	if methodCluster != "" && methodCluster != safeClusterName && methodCluster != constant.ClusterKeyFailfast {
		return false
	}
	for _, value := range []string{
		url.GetParam(constant.RetriesKey, ""),
		url.GetParam("methods."+method+"."+constant.RetriesKey, ""),
	} {
		if value == "" {
			continue
		}
		count, err := strconv.Atoi(value)
		if err != nil || count != 0 {
			return false
		}
	}
	return true
}
