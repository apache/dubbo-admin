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
	"os"

	"github.com/pkg/errors"
)

func TryWriteToDir(dir string) error {
	file, err := os.CreateTemp(dir, "write-access-check")
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, os.ModeDir|0o755); err != nil {
				return errors.Wrapf(err, "unable to create a directory %q", dir)
			}
			file, err = os.CreateTemp(dir, "write-access-check")
		}
		if err != nil {
			return errors.Wrapf(err, "unable to create temporary files in directory %q", dir)
		}
	}
	if err := os.Remove(file.Name()); err != nil {
		return errors.Wrapf(err, "unable to remove temporary files in directory %q", dir)
	}
	return nil
}
