// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"os"
	"sync"

	grpcZap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	mutex      = &sync.Mutex{}
	hasInit    = false
	cmdHasInit = false
	encoder    = zapcore.NewConsoleEncoder(
		zapcore.EncoderConfig{
			MessageKey:     "msg",
			LevelKey:       "level",
			TimeKey:        "time",
			CallerKey:      "line",
			NameKey:        "logger",
			FunctionKey:    "func",
			StacktraceKey:  "stacktrace",
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.0000"),
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
		})
	logger    *zap.Logger
	sugar     *zap.SugaredLogger
	cmdLogger *zap.Logger
	cmdSugar  *CmdSugarLogger
)

// CmdSugarLogger wraps zap.SugaredLogger and zapcore.WriteSyncer in order to use Sugar
// while being able to use low-level writers.
type CmdSugarLogger struct {
	*zap.SugaredLogger
	// wrap ws to print directly
	ws zapcore.WriteSyncer
}

func (log *CmdSugarLogger) Print(s string) {
	_, _ = log.ws.Write([]byte(s))
}

func Init() {
	mutex.Lock()
	defer mutex.Unlock()
	if hasInit {
		return
	}
	hasInit = true

	core := zapcore.NewCore(encoder, os.Stdout, zap.DebugLevel)
	logger = zap.New(core)
	defer logger.Sync() // flushes buffer, if any
	sugar = logger.Sugar()

	// Make sure that log statements internal to gRPC library are logged using the zapLogger as well.
	grpcZap.ReplaceGrpcLoggerV2(logger)
}

func InitCmdSugar(ws zapcore.WriteSyncer) {
	mutex.Lock()
	defer mutex.Unlock()
	if cmdHasInit {
		return
	}
	cmdHasInit = true

	core := zapcore.NewCore(encoder, ws, zap.DebugLevel)
	cmdLogger = zap.New(core)
	defer cmdLogger.Sync() // flushes buffer, if any
	cmdSugar = &CmdSugarLogger{
		SugaredLogger: cmdLogger.Sugar(),
		ws:            ws,
	}
}

func Sugar() *zap.SugaredLogger {
	if sugar == nil {
		Init()
	}
	return sugar
}

func Logger() *zap.Logger {
	if logger == nil {
		Init()
	}
	return logger
}

func CmdSugar() *CmdSugarLogger {
	if cmdSugar == nil {
		InitCmdSugar(os.Stdout)
	}
	return cmdSugar
}
