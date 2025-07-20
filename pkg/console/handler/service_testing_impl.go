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
	"context"

	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/util/reflection"
)

type TestingServiceImpl struct{}

func (t *TestingServiceImpl) GetMethodsNames(ctx context.Context, target, serviceName string) ([]string, error) {
	ref := reflection.NewRPCReflection(target)

	// dail the reflection server
	err := ref.Dail(ctx)
	if err != nil {
		return nil, err
	}
	defer ref.Close()

	// get methods
	methods, err := ref.ListMethods(serviceName)
	if err != nil {
		return nil, err
	}

	return methods, nil
}

// GetMethodDescribe get the detail  of method
func (t *TestingServiceImpl) GetMethodDescribe(ctx context.Context, target, methodName string) (*model.MethodDescribe, error) {
	ref := reflection.NewRPCReflection(target)

	// dail the reflection server
	err := ref.Dail(ctx)
	if err != nil {
		return nil, err
	}

	// get input and output type
	inputType, outputType, err := ref.InputAndOutputType(methodName)
	if err != nil {
		return nil, err
	}

	// get input and output describe string
	inputDescString, err := ref.DescribeString(inputType)
	if err != nil {
		return nil, err
	}
	outputDescString, err := ref.DescribeString(outputType)
	if err != nil {
		return nil, err
	}

	return &model.MethodDescribe{
		InputType:      inputType,
		InputDescribe:  inputDescString,
		OutputType:     outputType,
		OutputDescribe: outputDescString,
	}, nil
}

// GetMessageTemplateString get the template string of message
func (t *TestingServiceImpl) GetMessageTemplateString(ctx context.Context, target, messageName string) (string, error) {
	ref := reflection.NewRPCReflection(target)

	// dail the reflection server
	err := ref.Dail(ctx)
	if err != nil {
		return "", err
	}

	// get message template
	return ref.TemplateString(messageName)
}

// GetMessageDescribeString get the describe string (protobuf define string) of method
func (t *TestingServiceImpl) GetMessageDescribeString(ctx context.Context, target, messageName string) (string, error) {
	ref := reflection.NewRPCReflection(target)

	// dail the reflection server
	err := ref.Dail(ctx)
	if err != nil {
		return "", err
	}

	// get messageName describe
	return ref.DescribeString(messageName)
}

// Invoke the target method, input is json string,
// and return the response, success and error.
func (t *TestingServiceImpl) Invoke(ctx context.Context, target, methodName, input string, headers map[string]string) (string, error) {
	ref := reflection.NewRPCReflection(target)

	// dail the reflection server
	err := ref.Dail(ctx)
	if err != nil {
		return "", err
	}
	defer ref.Close()

	// set headers
	ref.SetRPCHeaders(headers)

	// invoke
	response, err := ref.Invoke(ctx, methodName, input)
	return response, err
}
