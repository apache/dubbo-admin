//go:build !windows
// +build !windows

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

package os

import (
	"fmt"
	"runtime"

	"golang.org/x/sys/unix"
)

func setFileLimit(n uint64) error {
	limit := unix.Rlimit{
		Cur: n,
		Max: n,
	}

	if err := unix.Setrlimit(unix.RLIMIT_NOFILE, &limit); err != nil {
		return fmt.Errorf("failed to set open file limit to %d: %w", limit.Cur, err)
	}

	return nil
}

// RaiseFileLimit raises the soft open file limit to match the hard limit.
func RaiseFileLimit() error {
	limit := unix.Rlimit{}
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &limit); err != nil {
		return fmt.Errorf("failed to query open file limits: %w", err)
	}

	// Darwin sets the max to unlimited, but it is actually limited
	// (typically to 24K) by the "kern.maxfilesperproc" systune.
	// Since we only run on Darwin for test purposes, just clip this
	// to a reasonable value.
	if runtime.GOOS == "darwin" && limit.Max > 10240 {
		limit.Max = 10240
	}

	return setFileLimit(limit.Max)
}

// CurrentFileLimit reports the current soft open file limit.
func CurrentFileLimit() (uint64, error) {
	limit := unix.Rlimit{}
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &limit); err != nil {
		return 0, fmt.Errorf("failed to query open file limits: %w", err)
	}

	return limit.Cur, nil
}
