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

package bizerror

type Error interface {
	Code() ErrorCode
	Message() string
	Error() string
	String() string
}

type ErrorCode string

const (
	UnknownError    ErrorCode = "UnknownError"
	InvalidArgument ErrorCode = "InvalidArgument"
	StoreError      ErrorCode = "StoreError"
	AppNotFound     ErrorCode = "AppNotFound"
	Unauthorized    ErrorCode = "Unauthorized"
	SessionError    ErrorCode = "SessionError"
)

type bizError struct {
	code    ErrorCode
	message string
}

var _ Error = &bizError{}

func NewBizError(code ErrorCode, message string) Error {
	return &bizError{
		code:    code,
		message: message,
	}
}

func (b *bizError) Code() ErrorCode {
	return b.code
}

func (b *bizError) Message() string {
	return b.message
}

func (b *bizError) Error() string {
	return b.String()
}

func (b *bizError) String() string {
	return string(b.code) + ": " + b.message
}
