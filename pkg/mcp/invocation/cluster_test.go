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
	"testing"

	"dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type countingInvoker struct {
	url      *common.URL
	attempts int
}

func (c *countingInvoker) GetURL() *common.URL { return c.url }
func (c *countingInvoker) IsAvailable() bool   { return true }
func (c *countingInvoker) Destroy()            {}
func (c *countingInvoker) Invoke(context.Context, base.Invocation) result.Result {
	c.attempts++
	return &result.RPCResult{Err: errors.New("provider failed")}
}

type candidateDirectory struct {
	url        *common.URL
	candidates []base.Invoker
	next       []base.Invoker
	lists      int
}

func (d *candidateDirectory) GetURL() *common.URL         { return d.url }
func (d *candidateDirectory) IsAvailable() bool           { return true }
func (d *candidateDirectory) Destroy()                    {}
func (d *candidateDirectory) Subscribe(*common.URL) error { return nil }
func (d *candidateDirectory) List(base.Invocation) []base.Invoker {
	d.lists++
	if d.lists > 1 && d.next != nil {
		return d.next
	}
	return d.candidates
}

type testClusterInterceptor struct{}

func (testClusterInterceptor) Invoke(ctx context.Context, next base.Invoker, call base.Invocation) result.Result {
	return next.Invoke(ctx, call)
}

func TestGlobalClusterInterceptorDoesNotWrapMCPAttempt(t *testing.T) {
	cluster.SetClusterInterceptor("mcp-test-observer", func() cluster.Interceptor {
		return testClusterInterceptor{}
	})

	refURL := common.NewURLWithOptions(
		common.WithProtocol(constant.DubboProtocol),
		common.WithParamsValue(constant.ClusterKey, safeClusterName),
		common.WithParamsValue(constant.RetriesKey, "0"),
	)
	regURL := common.NewURLWithOptions(common.WithProtocol(constant.RegistryProtocol))
	regURL.SubURL = refURL
	provider := &countingInvoker{url: refURL}
	dir := &candidateDirectory{url: regURL, candidates: []base.Invoker{provider}}

	invoker := safeCluster{}.Join(dir)
	err := invoker.Invoke(context.Background(), invocation.NewRPCInvocation(
		constant.Generic, []any{"update", []string{}, []any{}}, nil,
	)).Error()
	if err == nil || errors.Is(err, errUnsafeGovernance) || provider.attempts != 1 {
		t.Fatalf("interceptor-enabled call error=%v attempts=%d", err, provider.attempts)
	}
}

func TestGovernancePreflightAndSingleProviderAttempt(t *testing.T) {
	for _, protocol := range []string{constant.DubboProtocol, constant.TriProtocol} {
		t.Run(protocol, func(t *testing.T) {
			refURL := common.NewURLWithOptions(
				common.WithProtocol(protocol),
				common.WithParamsValue(constant.ClusterKey, safeClusterName),
				common.WithParamsValue(constant.RetriesKey, "0"),
			)
			regURL := common.NewURLWithOptions(common.WithProtocol(constant.RegistryProtocol))
			regURL.SubURL = refURL
			provider := &countingInvoker{url: common.NewURLWithOptions(
				common.WithProtocol(protocol),
				common.WithParamsValue(constant.ClusterKey, safeClusterName),
				common.WithParamsValue(constant.RetriesKey, "0"),
			)}
			dir := &candidateDirectory{url: regURL, candidates: []base.Invoker{provider}}
			safe := safeCluster{}.Join(dir)
			call := func() error {
				return safe.Invoke(context.Background(), invocation.NewRPCInvocation(
					constant.Generic, []any{"update", []string{}, []any{}}, nil,
				)).Error()
			}

			if err := call(); err == nil || errors.Is(err, errUnsafeGovernance) {
				t.Fatalf("admitted call returned %v", err)
			}
			if provider.attempts != 1 {
				t.Fatalf("admitted call attempted %d times, want 1", provider.attempts)
			}
			for _, setting := range []struct {
				name, key, value string
				url              *common.URL
			}{
				{"reference cluster", constant.ClusterKey, constant.ClusterKeyBroadcast, refURL},
				{"reference retries", constant.RetriesKey, "2", refURL},
				{"provider cluster", constant.ClusterKey, constant.ClusterKeyFailover, provider.url},
				{"provider retries", constant.RetriesKey, "3", provider.url},
				{"method retries", "methods.update.retries", "1", provider.url},
			} {
				t.Run(setting.name, func(t *testing.T) {
					old := setting.url.GetParam(setting.key, "")
					setting.url.SetParam(setting.key, setting.value)
					before := provider.attempts
					if err := call(); !errors.Is(err, errUnsafeGovernance) {
						t.Fatalf("unsafe override returned %v", err)
					}
					if provider.attempts != before {
						t.Fatalf("unsafe override dispatched %d attempts", provider.attempts-before)
					}
					setting.url.SetParam(setting.key, old)
				})
			}

			if err := call(); errors.Is(err, errUnsafeGovernance) {
				t.Fatalf("recovered governance returned %v", err)
			}
			if provider.attempts != 2 {
				t.Fatalf("recovered call attempted %d times, want 2 total", provider.attempts)
			}
		})
	}
}

func TestRouterIsEvaluatedOncePerCall(t *testing.T) {
	refURL := common.NewURLWithOptions(
		common.WithProtocol(constant.DubboProtocol),
		common.WithParamsValue(constant.ClusterKey, safeClusterName),
		common.WithParamsValue(constant.RetriesKey, "0"),
	)
	regURL := common.NewURLWithOptions(common.WithProtocol(constant.RegistryProtocol))
	regURL.SubURL = refURL
	safeProvider := &countingInvoker{url: refURL}
	unsafeProvider := &countingInvoker{url: common.NewURLWithOptions(
		common.WithProtocol(constant.DubboProtocol),
		common.WithParamsValue(constant.ClusterKey, constant.ClusterKeyBroadcast),
		common.WithParamsValue(constant.RetriesKey, "2"),
	)}
	dir := &candidateDirectory{url: regURL, candidates: []base.Invoker{safeProvider}, next: []base.Invoker{unsafeProvider}}
	safe := safeCluster{}.Join(dir)
	err := safe.Invoke(context.Background(), invocation.NewRPCInvocation(
		constant.Generic, []any{"update", []string{}, []any{}}, nil,
	)).Error()
	if err == nil || errors.Is(err, errUnsafeGovernance) {
		t.Fatalf("vetted provider call returned %v", err)
	}
	if dir.lists != 1 || safeProvider.attempts != 1 || unsafeProvider.attempts != 0 {
		t.Fatalf("directory lists=%d, safe attempts=%d, unsafe attempts=%d", dir.lists, safeProvider.attempts, unsafeProvider.attempts)
	}
}

func TestUnsafeOverrideBeforeClusterCreation(t *testing.T) {
	for _, protocol := range []string{constant.DubboProtocol, constant.TriProtocol} {
		for _, setting := range []struct{ key, value string }{
			{constant.ClusterKey, constant.ClusterKeyBroadcast},
			{constant.RetriesKey, "2"},
			{"methods.update.retries", "1"},
		} {
			refURL := common.NewURLWithOptions(
				common.WithProtocol(protocol),
				common.WithParamsValue(constant.ClusterKey, safeClusterName),
				common.WithParamsValue(constant.RetriesKey, "0"),
			)
			regURL := common.NewURLWithOptions(common.WithProtocol(constant.RegistryProtocol))
			regURL.SubURL = refURL
			provider := &countingInvoker{url: refURL}
			dir := &candidateDirectory{url: regURL, candidates: []base.Invoker{provider}}
			refURL.SetParam(setting.key, setting.value)
			safe := safeCluster{}.Join(dir)
			err := safe.Invoke(context.Background(), invocation.NewRPCInvocation(
				constant.Generic, []any{"update", []string{}, []any{}}, nil,
			)).Error()
			if !errors.Is(err, errUnsafeGovernance) || provider.attempts != 0 {
				t.Errorf("%s %s before creation: error=%v attempts=%d", protocol, setting.key, err, provider.attempts)
			}
		}
	}
}
