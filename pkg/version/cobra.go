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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

type Version struct {
	ClientVersion *BuildInfo `json:"clientVersion,omitempty" yaml:"clientVersion,omitempty"`
}

var (
	Info BuildInfo
)

func CobraCommandWithOptions() *cobra.Command {
	var (
		short   bool
		output  string
		version Version
	)
	ves := &cobra.Command{
		Use:   "version",
		Short: "Display the information of the currently built version",
		Long:  "The version command displays the information of the currently built version.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "" && output != "yaml" && output != "json" {
				return errors.New(`--output must be 'yaml' or 'json'`)
			}
			version.ClientVersion = &Info
			switch output {
			case "":
				if short {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "client version: %s\n", version.ClientVersion.Version)

				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "client version: %s\n", version.ClientVersion.LongForm())
				}
			case "yaml":
				if marshaled, err := yaml.Marshal(&version); err == nil {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(marshaled))
				}
			case "json":
				if marshaled, err := json.MarshalIndent(&version, "", "  "); err == nil {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(marshaled))
				}
			}

			return nil
		},
	}

	ves.Flags().BoolVarP(&short, "short", "s", false, "Use --short=false to generate full version information")
	ves.Flags().StringVarP(&output, "output", "o", "", "One of 'yaml' or 'json'.")
	return ves
}
