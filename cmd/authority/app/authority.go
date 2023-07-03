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

package app

import (
	"flag"
	"github.com/apache/dubbo-admin/pkg/authority"
	"github.com/apache/dubbo-admin/pkg/authority/config"
	"github.com/apache/dubbo-admin/pkg/logger"
	"github.com/spf13/cobra"
)

func NewAppCommand() *cobra.Command {
	options := config.NewOptions()

	cmd := &cobra.Command{
		Use:  "authority",
		Long: `The authority app is responsible for controllers in dubbo authority`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			logger.Infof("Authority command PersistentPreRun")
			if err := authority.Initialize(cmd); err != nil {
				logger.Fatal("Failed to initialize CA server.")
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Infof("Authority command Run with options: %+v", options)
			if errs := options.Validate(); len(errs) != 0 {
				logger.Fatal(errs)
				return errs[0]
			}

			if err := authority.Run(options); err != nil {
				logger.Fatal(err)
				return err
			}
			return nil
		},
	}

	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	options.FillFlags(cmd.Flags())
	return cmd
}
