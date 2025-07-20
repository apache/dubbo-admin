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
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	dubboVersion "github.com/apache/dubbo-kubernetes/pkg/version"
	"github.com/spf13/cobra"
)

func NewVersionCommand(_ cli.Context) *cobra.Command {
	var versionCmd *cobra.Command
	versionCmd = dubboVersion.CobraCommandWithOptions()
	return versionCmd
}
