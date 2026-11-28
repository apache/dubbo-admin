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
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	hessian "github.com/apache/dubbo-go-hessian2"

	dubbo "dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	dubbohessian "dubbo.apache.org/dubbo-go/v3/protocol/dubbo/hessian2"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	discovery "github.com/apache/dubbo-admin/pkg/config/discovery"
)

type Outcome string

const (
	NotExecuted    Outcome = "not-executed"
	UnknownOutcome Outcome = "unknown-outcome"
	RemoteError    Outcome = "remote-error"
)

var errNoProvider = errors.New("no provider available")

type CallError struct {
	Outcome  Outcome
	Protocol string
	Cause    error
}

func (e *CallError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Protocol, e.Outcome, e.Cause)
}

func (e *CallError) Unwrap() error { return e.Cause }

// Client caches one registry-backed reference for each mesh, registry,
// registration mode, service identity and selected wire protocol.
type Client struct {
	mu   sync.Mutex
	refs map[referenceKey]*reference
}

type referenceKey struct {
	mesh, registryID, registryURL, mode, service, group, version, protocol string
}

type reference struct {
	service *hessianService
	options *client.ReferenceOptions
}

// hessianService keeps the SDK-generated Invoke function, without exposing
// transport details to callers.
type hessianService struct {
	invoke func(context.Context, string, []string, []hessian.Object) (any, error)
}

func NewClient() *Client {
	return &Client{refs: make(map[referenceKey]*reference)}
}

// Call performs exactly one SDK generic invocation. Mode selects the native
// service or interface registry directory; it never selects a provider URL.
func (c *Client) Call(
	ctx context.Context,
	mesh string,
	config *discovery.Config,
	mode string,
	operation *meshproto.MCPOperation,
	args []any,
	timeout time.Duration,
) (any, error) {
	protocol := ""
	if operation != nil {
		protocol = operation.Protocol
	}
	reject := func(err error) (any, error) {
		return nil, &CallError{Outcome: NotExecuted, Protocol: protocol, Cause: err}
	}
	if ctx == nil || config == nil || operation == nil || mesh == "" ||
		config.ID == "" || operation.ServiceName == "" || operation.MethodName == "" {
		return reject(errors.New("mesh, discovery and operation are required"))
	}
	if config.Type != discovery.Zookeeper && config.Type != discovery.Nacos2 {
		return reject(fmt.Errorf("unsupported discovery type %q", config.Type))
	}
	if mode != constant.RegistryTypeService && mode != constant.RegistryTypeInterface {
		return reject(fmt.Errorf("unsupported registry mode %q", mode))
	}
	if protocol != constant.DubboProtocol && protocol != constant.TriProtocol {
		return reject(fmt.Errorf("unsupported protocol %q", protocol))
	}
	if len(args) != len(operation.ParameterTypes) || timeout <= 0 {
		return reject(errors.New("arguments and positive timeout are required"))
	}
	if err := ctx.Err(); err != nil {
		return reject(err)
	}
	key := referenceKey{
		mesh: mesh, registryID: config.ID, registryURL: config.Address.Registry,
		mode: mode, service: operation.ServiceName, group: operation.Group,
		version: operation.Version, protocol: protocol,
	}
	ref, err := c.getReference(key, config)
	if err != nil {
		return reject(err)
	}
	if ref.options == nil || ref.options.GetInvoker() == nil {
		return reject(errUnsafeGovernance)
	}
	if _, ok := ref.options.GetInvoker().(*safeInvoker); !ok {
		return reject(errUnsafeGovernance)
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	params := make([]hessian.Object, len(args))
	for i, arg := range args {
		params[i] = arg
	}
	type reply struct {
		value any
		err   error
	}
	completed := make(chan reply, 1)
	go func() {
		value, err := ref.service.invoke(callCtx, operation.MethodName, operation.ParameterTypes, params)
		completed <- reply{value: value, err: err}
	}()
	var response reply
	select {
	case response = <-completed:
	case <-callCtx.Done():
		return nil, &CallError{Outcome: UnknownOutcome, Protocol: protocol, Cause: callCtx.Err()}
	}
	value, err := response.value, response.err
	if callCtx.Err() != nil {
		return nil, &CallError{Outcome: UnknownOutcome, Protocol: protocol, Cause: callCtx.Err()}
	}
	if err == nil {
		return value, nil
	}
	if errors.Is(err, errUnsafeGovernance) || errors.Is(err, errNoProvider) {
		return reject(err)
	}
	var tripleError *tri.Error
	if protocol == constant.DubboProtocol && isDubboRemoteError(err) ||
		protocol == constant.TriProtocol && tri.IsWireError(err) &&
			errors.As(err, &tripleError) &&
			tripleError.Code() != tri.CodeDeadlineExceeded &&
			tripleError.Code() != tri.CodeCanceled &&
			tripleError.Code() != tri.CodeUnavailable {
		return nil, &CallError{Outcome: RemoteError, Protocol: protocol, Cause: err}
	}
	// Once Invoke starts, transport failures and deadlines can follow a
	// successful provider execution. Never retry or claim they did not run.
	return nil, &CallError{Outcome: UnknownOutcome, Protocol: protocol, Cause: err}
}

func isDubboRemoteError(err error) bool {
	for current := err; current != nil; current = errors.Unwrap(current) {
		if _, ok := dubbohessian.ToGenericException(current); ok {
			return true
		}
	}
	return false
}

func (c *Client) getReference(key referenceKey, config *discovery.Config) (*reference, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.refs == nil {
		c.refs = make(map[referenceKey]*reference)
	}
	if ref := c.refs[key]; ref != nil {
		return ref, nil
	}
	registryURL, err := url.Parse(config.Address.Registry)
	if err != nil || registryURL.Host == "" {
		return nil, fmt.Errorf("invalid discovery registry address")
	}
	if config.Type == discovery.Nacos2 && registryURL.Scheme != constant.NacosKey ||
		config.Type == discovery.Zookeeper && registryURL.Scheme != constant.ZookeeperKey {
		return nil, fmt.Errorf("discovery registry scheme does not match %s", config.Type)
	}
	registryOpts, err := registryOptions(registryURL, key.mode)
	if err != nil {
		return nil, err
	}
	ins, err := dubbo.NewInstance(dubbo.WithName("dubbo-admin-mcp"))
	if err != nil {
		return nil, err
	}
	clientOpts := []client.ClientOption{
		client.WithClientNoCheck(),
		client.WithClientRegistry(registryOpts...),
	}
	if key.protocol == constant.TriProtocol {
		clientOpts = append(clientOpts, client.WithClientProtocolTriple())
	} else {
		clientOpts = append(clientOpts, client.WithClientProtocolDubbo())
	}
	cli, err := ins.NewClient(clientOpts...)
	if err != nil {
		return nil, err
	}
	var options *client.ReferenceOptions
	svc, err := cli.NewGenericService(
		key.service,
		client.WithGroup(key.group),
		client.WithVersion(key.version),
		client.WithProtocol(key.protocol),
		client.WithCluster(safeClusterName),
		client.WithRetries(0),
		func(ref *client.ReferenceOptions) { options = ref },
	)
	if err != nil {
		return nil, err
	}
	ref := &reference{service: &hessianService{invoke: svc.Invoke}, options: options}
	c.refs[key] = ref
	return ref, nil
}

func registryOptions(registryURL *url.URL, mode string) ([]registry.Option, error) {
	opts := []registry.Option{
		registry.WithAddress(registryURL.Scheme + "://" + registryURL.Host + registryURL.Path),
		// The MCP consumer does not publish metadata; the SDK otherwise
		// starts a separate metadata-report connection for every reference.
		registry.WithoutUseAsMetaReport(),
	}
	query := registryURL.Query()
	params := make(map[string]string)
	for key, values := range query {
		if len(values) > 0 && values[0] != "" {
			params[key] = values[0]
		}
	}
	queryValue := func(keys ...string) string {
		for _, key := range keys {
			if value := query.Get(key); value != "" {
				return value
			}
		}
		return ""
	}
	removeParams := func(keys ...string) {
		for _, key := range keys {
			delete(params, key)
		}
	}

	username := queryValue("username")
	password := queryValue("password")
	if registryURL.User != nil {
		if username == "" {
			username = registryURL.User.Username()
		}
		if password == "" {
			password, _ = registryURL.User.Password()
		}
	}
	if username != "" {
		opts = append(opts, registry.WithUsername(username))
	}
	if password != "" {
		opts = append(opts, registry.WithPassword(password))
	}
	removeParams("username", "password")

	if value := queryValue("group", constant.RegistryGroupKey); value != "" {
		opts = append(opts, registry.WithGroup(value))
	}
	removeParams("group", constant.RegistryGroupKey)
	if value := queryValue("namespace", constant.RegistryNamespaceKey); value != "" {
		opts = append(opts, registry.WithNamespace(value))
	}
	removeParams("namespace", constant.RegistryNamespaceKey)

	if value := queryValue("timeout", constant.RegistryTimeoutKey); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return nil, fmt.Errorf("invalid registry timeout %q: %w", value, err)
		}
		opts = append(opts, registry.WithTimeout(timeout))
	}
	removeParams("timeout", constant.RegistryTimeoutKey)
	if value := queryValue("ttl", constant.RegistryTTLKey); value != "" {
		ttl, err := time.ParseDuration(value)
		if err != nil {
			return nil, fmt.Errorf("invalid registry ttl %q: %w", value, err)
		}
		opts = append(opts, registry.WithTTL(ttl))
	}
	removeParams("ttl", constant.RegistryTTLKey)
	if value := queryValue("simplified", constant.RegistrySimplifiedKey); value != "" {
		simplified, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid registry simplified flag %q: %w", value, err)
		}
		if simplified {
			opts = append(opts, registry.WithSimplified())
		}
	}
	removeParams("simplified", constant.RegistrySimplifiedKey)
	if value := queryValue("preferred", constant.PreferredKey); value != "" {
		preferred, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid registry preferred flag %q: %w", value, err)
		}
		if preferred {
			opts = append(opts, registry.WithPreferred())
		}
	}
	removeParams("preferred", constant.PreferredKey)
	if value := queryValue("zone", constant.RegistryZoneKey); value != "" {
		opts = append(opts, registry.WithZone(value))
	}
	removeParams("zone", constant.RegistryZoneKey)
	if value := queryValue("weight", constant.WeightKey); value != "" {
		weight, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid registry weight %q: %w", value, err)
		}
		opts = append(opts, registry.WithWeight(weight))
	}
	removeParams("weight", constant.WeightKey)

	if value := queryValue("accesskey", constant.RegistryAccessKey); value != "" {
		params[constant.RegistryAccessKey] = value
	}
	if value := queryValue("secretkey", constant.RegistrySecretKey); value != "" {
		params[constant.RegistrySecretKey] = value
	}
	removeParams("accesskey", "secretkey")
	if mode == constant.RegistryTypeInterface {
		opts = append(opts, registry.WithRegisterInterface())
	} else {
		opts = append(opts, registry.WithRegisterService())
	}
	if len(params) > 0 {
		opts = append(opts, registry.WithParams(params))
	}
	return opts, nil
}
