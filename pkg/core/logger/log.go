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
	"gopkg.in/natefinch/lumberjack.v2"

	logcfg "github.com/apache/dubbo-admin/pkg/config/log"
)

var (
	mutex          = &sync.Mutex{}
	hasInit        = false
	consoleEncoder = zapcore.NewConsoleEncoder(
		zapcore.EncoderConfig{
			MessageKey:     "msg",
			LevelKey:       "level",
			TimeKey:        "time",
			CallerKey:      "line",
			NameKey:        "logger",
			StacktraceKey:  "stacktrace",
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
		})
	jsonEncoder = zapcore.NewJSONEncoder(
		zapcore.EncoderConfig{
			MessageKey:     "msg",
			LevelKey:       "level",
			TimeKey:        "time",
			CallerKey:      "caller",
			NameKey:        "logger",
			StacktraceKey:  "stacktrace",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.FullCallerEncoder,
		})
	logger *zap.Logger
	sugar  *zap.SugaredLogger
)

var logLevelMap = map[logcfg.Level]zapcore.Level{
	logcfg.LevelDebug: zap.DebugLevel,
	logcfg.LevelInfo:  zap.InfoLevel,
	logcfg.LevelWarn:  zap.WarnLevel,
	logcfg.LevelError: zap.ErrorLevel,
}

func Init(cfg *logcfg.Config) {
	mutex.Lock()
	defer mutex.Unlock()
	if hasInit {
		return
	}
	hasInit = true

	var cores []zapcore.Core

	// output in terminal
	cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), logLevelMap[cfg.Level]))
	// output in file
	fileWriter := &lumberjack.Logger{
		Filename:   cfg.OutputPath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   true,
	}
	cores = append(cores, zapcore.NewCore(jsonEncoder, zapcore.AddSync(fileWriter), logLevelMap[cfg.Level]))
	combinedCore := zapcore.NewTee(cores...)

	logger = zap.New(combinedCore, zap.AddCaller(), zap.AddCallerSkip(2))
	defer logger.Sync() // flushes buffer, if any
	sugar = logger.Sugar()

	// Create a separate logger for gRPC with higher log level to suppress INFO logs
	grpcCore := zapcore.NewCore(consoleEncoder, os.Stdout, zap.WarnLevel)
	grpcLogger := zap.New(grpcCore, zap.AddCaller(), zap.AddCallerSkip(2))
	// Make sure that log statements internal to gRPC library are logged using the zapLogger as well.
	grpcZap.ReplaceGrpcLoggerV2(grpcLogger)
}

func Sugar() *zap.SugaredLogger {
	if sugar == nil {
		Init(logcfg.DefaultLogConfig())
	}
	return sugar
}

func Logger() *zap.Logger {
	if logger == nil {
		Init(logcfg.DefaultLogConfig())
	}
	return logger
}
