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

package cmd

import (
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/operator/cmd/cluster"
	"github.com/ory/viper"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

type createArgs struct {
	// dirname specifies the name of the custom-created directory.
	dirname string
	// language specifies different sdk languages.
	language string
	// template specifies repository or default common.
	template string
}

func addCreateFlags(cmd *cobra.Command, tempArgs *createArgs) {
	cmd.PersistentFlags().StringVarP(&tempArgs.language, "language", "l", "", "java or go language")
	cmd.PersistentFlags().StringVarP(&tempArgs.template, "template", "t", "", "java or go sdk template")
	cmd.PersistentFlags().StringVar(&tempArgs.dirname, "dirname", "", "java or go sdk template custom directory name")
}

type bindFunc func(*cobra.Command, []string) error

func bindEnv(flags ...string) bindFunc {
	return func(cmd *cobra.Command, args []string) (err error) {
		for _, flag := range flags {
			if err = viper.BindPFlag(flag, cmd.Flags().Lookup(flag)); err != nil {
				return
			}
		}
		viper.AutomaticEnv()
		return
	}
}

func CreateCmd(_ cli.Context, cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	rootArgs := &cluster.RootArgs{}
	tempArgs := &createArgs{}
	sc := sdkGenerateCmd(cmd, clientFactory)
	cc := &cobra.Command{
		Use:   "create",
		Short: "Create a custom dubbo sdk sample",
		Long:  "The create command will generates dubbo sdk.",
	}
	cluster.AddFlags(cc, rootArgs)
	cluster.AddFlags(sc, rootArgs)
	addCreateFlags(sc, tempArgs)
	cc.AddCommand(sc)
	return cc
}

func sdkGenerateCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "sdk",
		Short: "Generate sdk samples for Dubbo supported languages",
		Long:  "The sdk subcommand generates an sdk sample provided by Dubbo supported languages.",
		Example: `  # Create a java sample sdk.
  dubboctl create sdk --language java --template common --dirname mydubbo

  # Select a default java repository.
  dubboctl create sdk -l java -t common --dirname mydubbo

  # Select a java repository.
  dubboctl create sdk -l java -t myrepo/mydubbo --dirname myapp

  # Create a go sample sdk.
  dubboctl create sdk --language go --template common --dirname mydubbogo
  
  # Select a default go repository.
  dubboctl create sdk -l go -t common --dirname mydubbogo

  # Select a go repository.
  dubboctl create sdk -l go -t myrepo/mydubbo --dirname myapp 
`,
		PreRunE: bindEnv("language", "template", "dirname"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, args, clientFactory)
		},
	}
}

type createConfig struct {
	// Path Absolute to function source
	Path     string
	Runtime  string
	Template string
	// Repo Uri (overrides builtin and installed)
	Repo string
	// DirName Defines a custom creation directory name。
	DirName     string
	Initialized bool
}

// newCreateConfig returns a config populated from the current execution context
// (args, flags and environment variables)
// The client constructor function is used to create a transient client for
// accessing things like the current valid templates list, and uses the
// current value of the config at time of prompting.
func newCreateConfig(_ *cobra.Command, _ []string, _ ClientFactory) (cc createConfig, err error) {
	var absolutePath string
	absolutePath = cwd()

	cc = createConfig{
		DirName:     viper.GetString("dirname"),
		Path:        absolutePath + "/" + viper.GetString("dirname"),
		Runtime:     viper.GetString("language"),
		Template:    viper.GetString("template"),
		Initialized: viper.GetBool("initialized"),
	}
	fmt.Printf("Name:     %v\n", cc.DirName)
	fmt.Printf("Path:     %v\n", cc.Path)
	fmt.Printf("Language:     %v\n", cc.Runtime)
	fmt.Printf("Template:     %v\n", cc.Template)
	return
}

func runCreate(cmd *cobra.Command, args []string, clientFactory ClientFactory) error {
	// Create a config based on args.  Also uses the newClient to create a
	// temporary client for completing options such as available runtimes.
	createCfg, err := newCreateConfig(cmd, args, clientFactory)
	if err != nil {
		return err
	}
	// From environment variables, flags, arguments, and user prompts if --confirm
	// (in increasing levels of precedence)
	client, cancel := clientFactory()
	defer cancel()

	// a deeper validation than that which is performed when
	// instantiating the client with the raw config above.
	if err = createCfg.validate(client); err != nil {
		return err
	}

	// Initialization creation
	_, err = client.Initialize(&dubbo.DubboConfig{
		Root:     createCfg.Path,
		Name:     createCfg.DirName,
		Runtime:  createCfg.Runtime,
		Template: createCfg.Template,
	}, createCfg.Initialized, cmd)
	if err != nil {
		return err
	}

	fmt.Printf("dubbo %v sdk was successfully created.\n", createCfg.Runtime)
	return nil
}

type ErrNoRuntime error
type ErrInvalidRuntime error
type ErrInvalidTemplate error

func (c createConfig) validate(client *sdk.Client) (err error) {
	if c.Runtime == "" {
		return noRuntimeError(client)
	}

	if c.Runtime != "" && c.Repo == "" &&
		!isValidRuntime(client, c.Runtime) {
		return newInvalidRuntimeError(client, c.Runtime)
	}

	if c.Template != "" && c.Repo == "" &&
		!isValidTemplate(client, c.Runtime, c.Template) {
		return newInvalidTemplateError(client, c.Runtime, c.Template)
	}

	return
}

func newInvalidRuntimeError(client *sdk.Client, runtime string) error {
	b := strings.Builder{}
	fmt.Fprintf(&b, "The language runtime '%v' is not recognized.\n", runtime)
	runtimes, err := client.Runtimes()
	if err != nil {
		return err
	}
	for _, v := range runtimes {
		fmt.Fprintf(&b, "  %v\n", v)
	}
	return ErrInvalidRuntime(errors.New(b.String()))
}

// isValidTemplate determines if the given template is valid for the given runtime.
func isValidTemplate(client *sdk.Client, runtime, template string) bool {
	if !isValidRuntime(client, runtime) {
		return false
	}
	templates, err := client.Templates().List(runtime)
	if err != nil {
		return false
	}
	for _, v := range templates {
		if v == template {
			return true
		}
	}
	return false
}

// isValidRuntime determines if the given language runtime is a valid choice.
func isValidRuntime(client *sdk.Client, runtime string) bool {
	runtimes, err := client.Runtimes()
	if err != nil {
		return false
	}
	for _, v := range runtimes {
		if v == runtime {
			return true
		}
	}
	return false
}

func noRuntimeError(client *sdk.Client) error {
	b := strings.Builder{}
	fmt.Fprintln(&b, "Required flag \"language\" not set.")
	fmt.Fprintln(&b, "Available language runtimes are:")
	runtimes, err := client.Runtimes()
	if err != nil {
		return err
	}
	for _, v := range runtimes {
		fmt.Fprintf(&b, "  %v\n", v)
	}
	return ErrNoRuntime(errors.New(b.String()))
}

func newInvalidTemplateError(client *sdk.Client, runtime, template string) error {
	b := strings.Builder{}
	fmt.Fprintf(&b, "The template '%v' was not found for language runtime '%v'.\n", template, runtime)
	fmt.Fprintln(&b, "Available templates for this language runtime are:")
	templates, err := client.Templates().List(runtime)
	if err != nil {
		return err
	}
	for _, v := range templates {
		fmt.Fprintf(&b, "  %v\n", v)
	}
	return ErrInvalidTemplate(errors.New(b.String()))
}

func cwd() (cwd string) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("Unable to determine current working directory: %v", err))
	}
	return cwd
}
