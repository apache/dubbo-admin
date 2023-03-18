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
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/apache/dubbo-admin/pkg/authority/config"
	"github.com/apache/dubbo-admin/pkg/authority/security"
	"github.com/apache/dubbo-admin/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	// For example, --webhook-port is bound to DUBBO_WEBHOOK_PORT.
	envNamePrefix = "DUBBO"

	// Replace hyphenated flag names with camelCase
	replaceWithCamelCase = false
)

func NewAppCommand() *cobra.Command {
	options := config.NewOptions()

	cmd := &cobra.Command{
		Use:  "authority",
		Long: `The authority app is responsible for controllers in dubbo authority`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logger.Sugar().Infof("Authority command PersistentPreRun")
			initialize(cmd)
		},
		Run: func(cmd *cobra.Command, args []string) {
			logger.Sugar().Infof("Authority command Run with options: %+v", options)
			if errs := options.Validate(); len(errs) != 0 {
				log.Fatal(errs)
			}

			if err := Run(options); err != nil {
				log.Fatal(err)
			}
		},
	}

	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	options.FillFlags(cmd.Flags())
	return cmd
}

func Run(options *config.Options) error {
	s := security.NewServer(options)

	s.Init()
	s.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(s.StopChan, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(s.CertStorage.GetStopChan(), syscall.SIGINT, syscall.SIGTERM)

	<-c

	return nil
}

func initialize(cmd *cobra.Command) error {
	v := viper.New()

	// For example, --webhook-port is bound to DUBBO_WEBHOOK_PORT.
	v.SetEnvPrefix(envNamePrefix)

	// keys with underscores, e.g. DUBBO-WEBHOOK-PORT to DUBBO_WEBHOOK_PORT
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Bind to environment variables
	v.AutomaticEnv()

	// Bind the current command's flags to viper
	bindFlags(cmd, v)

	return nil
}

func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		configName := f.Name

		//  Replace hyphens with a camelCased string.
		if replaceWithCamelCase {
			configName = strings.ReplaceAll(f.Name, "-", "")
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}
