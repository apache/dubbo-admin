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

package log

import (
	"fmt"

	"github.com/duke-git/lancet/v2/slice"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/config"
)

type Level string

var supportLevels = []Level{LevelDebug, LevelInfo, LevelWarn, LevelError}

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

const (
	defaultLogLevel   = LevelDebug
	defaultOutputPath = "logs/dubbo-admin.log"
	defaultMaxSize    = 100
	defaultMaxBackups = 5
	defaultMaxAge     = 3
)

type Config struct {
	config.BaseConfig
	Level      Level  `json:"level" yaml:"level"`
	OutputPath string `json:"outputPath" yaml:"outputPath"`
	MaxSize    int    `json:"maxSize" yaml:"maxSize"`
	MaxBackups int    `json:"maxBackups" yaml:"maxBackups"`
	MaxAge     int    `json:"maxAge" yaml:"maxAge"`
}

func (c *Config) Validate() error {
	if !slice.Contain(supportLevels, c.Level) {
		return bizerror.New(bizerror.ConfigError, fmt.Sprintf("unsupported log level: %s", c.Level))
	}
	return nil
}

func DefaultLogConfig() *Config {
	return &Config{
		Level:      defaultLogLevel,
		OutputPath: defaultOutputPath,
		MaxSize:    defaultMaxSize,
		MaxBackups: defaultMaxBackups,
		MaxAge:     defaultMaxAge,
	}
}
