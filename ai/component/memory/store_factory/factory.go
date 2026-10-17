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

package storefactory

import (
	"fmt"
	"io"

	conversationstore "dubbo-admin-ai/store"
)

type DatabaseConfig struct {
	Driver       string
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
}

type Config struct {
	Backend  string
	MaxTurns int
	Database *DatabaseConfig
}

type opener func(Config) (conversationstore.Store, io.Closer, error)

// Open selects a storage implementation while the component keeps only the
// shared Store contract and an optional closer.
func Open(config Config) (conversationstore.Store, io.Closer, error) {
	if config.Backend == "" {
		config.Backend = "memory"
	}
	openers := map[string]opener{
		"memory": openMemory,
		"gorm":   openGorm,
	}
	open, ok := openers[config.Backend]
	if !ok {
		return nil, nil, fmt.Errorf("unsupported memory backend %q", config.Backend)
	}
	return open(config)
}
