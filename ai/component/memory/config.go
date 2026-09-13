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

package memory

import (
	"fmt"
	"strings"
)

const (
	DefaultBackend      = "memory"
	DefaultMaxOpenConns = 100
	DefaultMaxIdleConns = 10
)

type MemorySpec struct {
	Backend    string        `yaml:"backend"`
	HistoryKey HistoryKey    `yaml:"history_key"` // History key name
	MaxTurns   int           `yaml:"max_turns"`   // Maximum conversation turns
	Database   *DatabaseSpec `yaml:"database"`
}

type DatabaseSpec struct {
	Driver       string `yaml:"driver"`
	DSN          string `yaml:"dsn"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

func DefaultMemorySpec() *MemorySpec {
	return &MemorySpec{
		Backend:    DefaultBackend,
		HistoryKey: ChatHistoryKey,
		MaxTurns:   100,
	}
}

func (s MemorySpec) Validate() error {
	backend := strings.ToLower(strings.TrimSpace(s.Backend))
	if backend == "" {
		backend = DefaultBackend
	}
	if backend != "memory" && backend != "gorm" {
		return fmt.Errorf("unsupported memory backend %q", s.Backend)
	}
	if s.MaxTurns <= 0 {
		return fmt.Errorf("max_turns must be greater than 0")
	}
	if s.Database == nil {
		if backend == "gorm" {
			return fmt.Errorf("database is required for gorm backend")
		}
		return nil
	}
	if backend != "gorm" {
		return nil
	}

	database := *s.Database
	database.applyDefaults()
	if strings.ToLower(strings.TrimSpace(database.Driver)) != "mysql" && strings.ToLower(strings.TrimSpace(database.Driver)) != "postgres" {
		return fmt.Errorf("unsupported database driver %q", s.Database.Driver)
	}
	if strings.TrimSpace(database.DSN) == "" {
		return fmt.Errorf("database dsn is required")
	}
	if database.MaxOpenConns <= 0 {
		return fmt.Errorf("max_open_conns must be greater than 0")
	}
	if database.MaxIdleConns <= 0 {
		return fmt.Errorf("max_idle_conns must be greater than 0")
	}
	if database.MaxIdleConns > database.MaxOpenConns {
		return fmt.Errorf("max_idle_conns must not exceed max_open_conns")
	}
	return nil
}

func (d *DatabaseSpec) applyDefaults() {
	if d.MaxOpenConns == 0 {
		d.MaxOpenConns = DefaultMaxOpenConns
	}
	if d.MaxIdleConns == 0 {
		d.MaxIdleConns = DefaultMaxIdleConns
	}
}
