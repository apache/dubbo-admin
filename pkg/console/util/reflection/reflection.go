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

package reflection

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type RPCReflection interface {
	// SetUserAgent sets the User-Agent header to be sent in each request.
	SetUserAgent(userAgent string)
	// SetConnectTimeout sets the timeout for dialing a target.
	SetConnectTimeout(timeout time.Duration)
	// SetKeepaliveTime sets the keepalive time for grpc connection.
	SetKeepaliveTime(timeout time.Duration)
	// SetAdditionalHeaders sets the additional headers to be sent in both reflection request and rpc request.
	SetAdditionalHeaders(headers map[string]string)
	// SetReflectionHeaders sets the additional headers to be sent in only reflection request.
	SetReflectionHeaders(headers map[string]string)
	// SetRPCHeaders sets the additional headers to be sent in only rpc request.
	SetRPCHeaders(headers map[string]string)
	// Dail to the target, you should call this method before send reflection request and rpc request
	Dail(ctx context.Context) error
	// Close the connection.
	Close()
	// ListServices returns all services in the target.
	ListServices() ([]string, error)
	// ListMethods returns all methods in the service.
	ListMethods(service string) ([]string, error)
	// Invoke invokes the method with input.
	Invoke(ctx context.Context, methodName, input string) (response string, err error)
	// TemplateString returns the template string of the message.
	TemplateString(messageName string) (string, error)
	// DescribeString returns the description string of the message.
	DescribeString(symbol string) (string, error)
	// Descriptor returns the desc.Descriptor.
	Descriptor(symbol string) (desc.Descriptor, error)
	// InputAndOutputType returns the input and output type of the method.
	InputAndOutputType(methodName string) (string, string, error)
}

type rpcReflection struct {
	target         string // remote address
	userAgent      string
	connectTimeout time.Duration
	keepaliveTime  time.Duration

	// additionalHeaders will be included in both reflection request and rpc request
	additionalHeaders map[string]string
	// reflectionHeaders will be included in only the reflection request
	reflectionHeaders map[string]string
	// rpcHeaders will be included in only the rpc request
	rpcHeaders map[string]string

	mu         sync.Mutex // to protect descSource and clientConn
	refClient  *grpcreflect.Client
	clientConn *grpc.ClientConn
	descSource grpcurl.DescriptorSource
}

func NewRPCReflection(target string) RPCReflection {
	r := &rpcReflection{
		target: target,
	}

	return r
}

func (r *rpcReflection) SetUserAgent(userAgent string) {
	r.userAgent = userAgent
}

func (r *rpcReflection) SetConnectTimeout(timeout time.Duration) {
	r.connectTimeout = timeout
}

func (r *rpcReflection) SetKeepaliveTime(timeout time.Duration) {
	r.keepaliveTime = timeout
}

func (r *rpcReflection) SetAdditionalHeaders(headers map[string]string) {
	r.additionalHeaders = headers
}

func (r *rpcReflection) SetReflectionHeaders(headers map[string]string) {
	r.reflectionHeaders = headers
}

func (r *rpcReflection) SetRPCHeaders(headers map[string]string) {
	r.rpcHeaders = headers
}

// Dail to the target, you should call this method before send reflection request and rpc request
func (r *rpcReflection) Dail(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.clientConn != nil && r.descSource != nil {
		return nil
	}

	md := grpcurl.MetadataFromHeaders(append(headerMapToStrings(r.additionalHeaders), headerMapToStrings(r.reflectionHeaders)...))
	refCtx := metadata.NewOutgoingContext(ctx, md)

	dail := func(ctx context.Context) (*grpc.ClientConn, error) {
		// add time out and keep alive
		dialTime := 10 * time.Second
		if r.connectTimeout > 0 {
			dialTime = r.connectTimeout
		}
		ctx, cancel := context.WithTimeout(ctx, dialTime)
		defer cancel()
		var opts []grpc.DialOption
		if r.keepaliveTime > 0 {
			timeout := r.keepaliveTime
			opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:    timeout,
				Timeout: timeout,
			}))
		}

		var creds credentials.TransportCredentials

		// add user agent
		opts = append(opts, grpc.WithUserAgent(r.userAgent))

		network := "tcp"
		cc, err := grpcurl.BlockingDial(ctx, network, r.target, creds, opts...)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to dial target host %s", r.target)
		}

		return cc, nil
	}

	cc, err := dail(refCtx)
	if err != nil {
		return err
	}

	r.clientConn = cc
	reflectionClient := grpcreflect.NewClientAuto(refCtx, cc)
	r.refClient = reflectionClient
	reflectionSource := grpcurl.DescriptorSourceFromServer(ctx, reflectionClient)
	r.descSource = reflectionSource

	return nil
}

func (r *rpcReflection) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.refClient != nil {
		r.refClient.Reset()
		r.refClient = nil
	}

	if r.clientConn != nil {
		r.clientConn.Close()
		r.clientConn = nil
	}
}

func (r *rpcReflection) ListServices() ([]string, error) {
	svcs, err := grpcurl.ListServices(r.descSource)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list services")
	}

	return svcs, nil
}

func (r *rpcReflection) ListMethods(service string) ([]string, error) {
	methods, err := grpcurl.ListMethods(r.descSource, service)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to list methods for service %s", service)
	}

	return methods, nil
}

func headerMapToStrings(headerMap map[string]string) []string {
	var headers []string
	for k, v := range headerMap {
		headers = append(headers, fmt.Sprintf("%v: %v", k, v))
	}
	return headers
}

func (r *rpcReflection) Invoke(ctx context.Context, methodName, input string) (response string, err error) {
	// Invoke an RPC
	cc := r.clientConn

	// input string, request message
	var in io.Reader
	in = strings.NewReader(input)

	options := grpcurl.FormatOptions{
		EmitJSONDefaultFields: true,
		IncludeTextSeparator:  true,
		AllowUnknownFields:    true,
	}
	rf, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, r.descSource, in, options)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to construct request parser and formatter for %v", err)
	}

	// invoke
	output := bytes.NewBuffer(nil)
	h := &grpcurl.DefaultEventHandler{
		Out:            output,
		Formatter:      formatter,
		VerbosityLevel: 0, // no verbose
	}

	addlHeaders := headerMapToStrings(r.additionalHeaders)
	rpcHeaders := headerMapToStrings(r.rpcHeaders)
	err = grpcurl.InvokeRPC(ctx, r.descSource, cc, methodName, append(addlHeaders, rpcHeaders...), h, rf.Next)
	if err != nil {
		if errStatus, ok := status.FromError(err); ok {
			h.Status = errStatus
		}

		return "", err
	}

	if h.Status.Code() != codes.OK {
		// failed to invoke
		formattedStatus, err := formatter(h.Status.Proto())
		if err != nil {
			return "", nil
		}
		return "", errors.New(formattedStatus)
	}

	// success invoke
	return output.String(), nil
}

func (r *rpcReflection) TemplateString(messageName string) (string, error) {
	dsc, err := r.Descriptor(messageName)
	if err != nil {
		return "", err
	}

	msgDesc, ok := dsc.(*desc.MessageDescriptor)
	if !ok {
		return "", errors.New("not a message")
	}
	// for messages, also show a template in JSON, to make it easier to
	// create a request to invoke an RPC
	tmpl := grpcurl.MakeTemplate(msgDesc)
	options := grpcurl.FormatOptions{EmitJSONDefaultFields: true}
	_, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, r.descSource, nil, options)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to construct formatter, err=%v", err)
	}
	template, err := formatter(tmpl)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to print template for message %s", messageName)
	}

	return template, nil
}

func (r *rpcReflection) DescribeString(symbol string) (string, error) {
	dsc, err := r.Descriptor(symbol)
	if err != nil {
		return "", err
	}

	txt, err := grpcurl.GetDescriptorText(dsc, r.descSource)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to describe symbol %q", symbol)
	}

	return txt, nil
}

func (r *rpcReflection) Descriptor(symbol string) (desc.Descriptor, error) {
	if symbol[0] == '.' {
		symbol = symbol[1:]
	}

	dsc, err := r.descSource.FindSymbol(symbol)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to resolve symbol %q", symbol)
	}

	return dsc, nil
}

func (r *rpcReflection) InputAndOutputType(methodName string) (string, string, error) {
	// get descriptor of method
	descriptor, err := r.Descriptor(methodName)
	if err != nil {
		return "", "", err
	}

	methodDesc, ok := descriptor.(*desc.MethodDescriptor)
	if !ok {
		return "", "", fmt.Errorf("%s is not a method", methodName)
	}

	// get input and output descriptor
	inputDesc := methodDesc.GetInputType()
	outputDesc := methodDesc.GetOutputType()

	return inputDesc.GetFullyQualifiedName(), outputDesc.GetFullyQualifiedName(), nil
}
