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

package xds

import (
	"fmt"

	envoy_log "github.com/envoyproxy/go-control-plane/pkg/log"
	"github.com/go-logr/logr"
)

func NewLogger(log logr.Logger) envoy_log.Logger {
	return &logger{log: log}
}

type logger struct {
	log logr.Logger
}

func (l logger) Debugf(format string, args ...interface{}) {
	l.log.V(1).Info(fmt.Sprintf(format, args...))
}

func (l logger) Warnf(format string, args ...interface{}) {
	l.log.V(1).Info(fmt.Sprintf(format, args...))
}

func (l logger) Infof(format string, args ...interface{}) {
	l.log.V(1).Info(fmt.Sprintf(format, args...))
}

func (l logger) Errorf(format string, args ...interface{}) {
	l.log.Error(fmt.Errorf(format, args...), "")
}
