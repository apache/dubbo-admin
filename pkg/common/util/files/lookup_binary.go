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

package files

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

type LookupPathFn = func() (string, error)

// LookupNextToCurrentExecutable looks for the binary next to the current binary
// Example: if this function is executed by /usr/bin/dubbo-dp, this function will lookup for binary 'x' in /usr/bin/x
func LookupNextToCurrentExecutable(binary string) LookupPathFn {
	return func() (string, error) {
		ex, err := os.Executable()
		if err != nil {
			return "", err
		}
		return filepath.Dir(ex) + "/" + binary, nil
	}
}

// LookupInCurrentDirectory looks for the binary in the current directory
// Example: if this function is executed by /usr/bin/dubbo-dp that was run in /home/dubbo-dp, this function will lookup for binary 'x' in /home/dubbo-dp/x
func LookupInCurrentDirectory(binary string) LookupPathFn {
	return func() (string, error) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return cwd + "/" + binary, nil
	}
}

func LookupInPath(path string) LookupPathFn {
	return func() (string, error) {
		return path, nil
	}
}

// LookupBinaryPath looks for a binary in order of passed lookup functions.
// It fails only if all lookup function does not contain a binary.
func LookupBinaryPath(pathFns ...LookupPathFn) (string, error) {
	var candidatePaths []string
	for _, candidatePathFn := range pathFns {
		candidatePath, err := candidatePathFn()
		if err != nil {
			continue
		}
		candidatePaths = append(candidatePaths, candidatePath)
		path, err := exec.LookPath(candidatePath)
		if err == nil {
			return path, nil
		}
	}

	return "", errors.Errorf("could not find executable binary in any of the following paths: %v", candidatePaths)
}
