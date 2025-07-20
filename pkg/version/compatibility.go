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

package version

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/apache/dubbo-admin/pkg/core"
)

var log = core.Log.WithName("version").WithName("compatibility")

var PreviewVersionPrefix = "preview"

func IsPreviewVersion(version string) bool {
	return strings.Contains(version, PreviewVersionPrefix)
}

// DeploymentVersionCompatible returns true if the given component version
// is compatible with the installed version of dubbo CP.
// For all binaries which share a common version (dubbo DP, CP, Zone CP...), we
// support backwards compatibility of at most two prior minor versions.
func DeploymentVersionCompatible(dubboVersionStr, componentVersionStr string) bool {
	if IsPreviewVersion(dubboVersionStr) || IsPreviewVersion(componentVersionStr) {
		return true
	}

	dubboVersion, err := semver.NewVersion(dubboVersionStr)
	if err != nil {
		// Assume some kind of dev version
		log.Info("cannot parse semantic version", "version", dubboVersionStr)
		return true
	}

	componentVersion, err := semver.NewVersion(componentVersionStr)
	if err != nil {
		// Assume some kind of dev version
		log.Info("cannot parse semantic version", "version", componentVersionStr)
		return true
	}

	minMinor := int64(dubboVersion.Minor()) - 2
	if minMinor < 0 {
		minMinor = 0
	}

	maxMinor := dubboVersion.Minor() + 2

	constraint, err := semver.NewConstraint(
		fmt.Sprintf(">= %d.%d, <= %d.%d", dubboVersion.Major(), minMinor, dubboVersion.Major(), maxMinor),
	)
	if err != nil {
		// Programmer error
		panic(err)
	}

	return constraint.Check(componentVersion)
}
