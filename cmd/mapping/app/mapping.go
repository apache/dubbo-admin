/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"flag"
	"fmt"
	"github.com/apache/dubbo-admin/pkg/logger"
	"github.com/apache/dubbo-admin/pkg/mapping/bootstrap"
	"github.com/apache/dubbo-admin/pkg/mapping/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	replaceWithCamelCase = false
)

func Flags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		configName := f.Name
		if replaceWithCamelCase {
			configName = strings.ReplaceAll(f.Name, "-", "")
		}
		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}

func initial(cmd *cobra.Command) error {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()
	Flags(cmd, v)
	return nil
}

func Command() *cobra.Command {
	options := config.NewOptions()
	cmd := &cobra.Command{
		Use: "ServiceMapping",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logger.Infof("PreRun Service Mapping")
			initial(cmd)
		},
		Run: func(cmd *cobra.Command, args []string) {
			logger.Infof("Run Service Mapping %+v", options)
			if err := Run(options); err != nil {
				logger.Fatal(err)
			}
		},
	}
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	options.FillFlags(cmd.Flags())
	return cmd
}

func Run(option *config.Options) error {
	s := bootstrap.NewServer(option)

	s.Init()
	s.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(s.StopChan, syscall.SIGINT, syscall.SIGTERM)
	<-c
	return nil
}
