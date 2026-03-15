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

package logger

import (
	"dubbo-admin-ai/runtime"
	"fmt"
	"log/slog"

	"github.com/dusted-go/logging/prettylog"
)

// LoggerComponent implements the logger component
type LoggerComponent struct {
	instanceName string
	// Runtime configuration fields (converted from cfg, original cfg not retained)
	level string
}

// NewLoggerComponent creates a logger component instance
func NewLoggerComponent(level string) (runtime.Component, error) {
	return &LoggerComponent{
		level: level,
	}, nil
}

// Name returns the component name
func (l *LoggerComponent) Name() string {
	if l.instanceName != "" {
		return l.instanceName
	}
	return "logger"
}

func (l *LoggerComponent) SetName(name string) {
	l.instanceName = name
}

func (l *LoggerComponent) Validate() error {
	switch l.level {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("invalid logger level: %s", l.level)
	}
}

func (l *LoggerComponent) Init(rt *runtime.Runtime) error {
	var logger *slog.Logger
	switch l.level {
	case "debug":
		logger = l.debugLogger()
	case "info":
		logger = l.infoLogger()
	default:
		logger = l.infoLogger()
	}
	slog.SetDefault(logger)
	slog.Info("Logger component initialized",
		"level", l.level)

	return nil
}

func (l *LoggerComponent) Start() error {
	return nil
}

func (l *LoggerComponent) Stop() error {
	return nil
}

func (l *LoggerComponent) debugLogger() *slog.Logger {
	slog.SetDefault(
		slog.New(
			prettylog.NewHandler(&slog.HandlerOptions{
				Level:       slog.LevelDebug,
				AddSource:   true,
				ReplaceAttr: nil,
			}),
		),
	)
	return slog.Default()
}

func (l *LoggerComponent) infoLogger() *slog.Logger {
	slog.SetDefault(
		slog.New(
			prettylog.NewHandler(&slog.HandlerOptions{
				Level:       slog.LevelInfo,
				AddSource:   false,
				ReplaceAttr: nil,
			}),
		),
	)
	return slog.Default()
}
