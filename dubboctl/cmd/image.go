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
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/builder/dockerfile"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/builder/pack"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"github.com/ory/viper"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type imageArgs struct {
	// dockerfile Defines the required Dockerfile files.
	dockerfile bool
	// builder Defines the path used by the builder.
	builder bool
	// output Defines the generated dubbo deploy yaml file.
	output string
	// destroy Defines the deletion of deploy yaml file.
	destroy bool
}

func addHubFlags(cmd *cobra.Command, iArgs *imageArgs) {
	cmd.PersistentFlags().BoolVarP(&iArgs.dockerfile, "file", "f", false, "Specify the file as a dockerfile")
	cmd.PersistentFlags().BoolVarP(&iArgs.builder, "builder", "b", false, "The builder generates the image")
}

func addDeployFlags(cmd *cobra.Command, iArgs *imageArgs) {
	cmd.PersistentFlags().StringVarP(&iArgs.output, "output", "o", "dubbo-deploy.yaml", "The output generates k8s yaml file")
	cmd.PersistentFlags().BoolVarP(&iArgs.destroy, "delete", "d", false, "deletion k8s yaml file")
}

func ImageCmd(ctx cli.Context, cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	ihc := imageHubCmd(cmd, clientFactory)
	idc := imageDeployCmd(cmd, clientFactory)
	ic := &cobra.Command{
		Use:   "image",
		Short: "Used to build and push images, apply to cluster",
	}
	ic.AddCommand(ihc)
	ic.AddCommand(idc)
	return ic
}

type hubConfig struct {
	// Dockerfile Defines the required files.
	Dockerfile bool
	// Builder Defines the path used by the builder.
	Builder bool
	// Image information required by image.
	Image string
	// BuilderImage is the image (name or mapping) to use for building.  Usually
	// set automatically.
	BuilderImage string
	// Path of the application implementation on local disk. Defaults to current
	// working directory of the process.
	Path string
}

type deployConfig struct {
	*hubConfig
	Output    string
	Destroy   bool
	Namespace string
	Port      int
	Path      string
}

func newHubConfig(cmd *cobra.Command) *hubConfig {
	hc := &hubConfig{
		Dockerfile: viper.GetBool("file"),
		Builder:    viper.GetBool("builder"),
	}
	return hc
}

func newDeployConfig(cmd *cobra.Command) *deployConfig {
	dc := &deployConfig{
		hubConfig: newHubConfig(cmd),
		Output:    viper.GetString("output"),
		Destroy:   viper.GetBool("delete"),
	}
	return dc
}

func (hc hubConfig) imageClientOptions() ([]sdk.Option, error) {
	var do []sdk.Option
	if hc.Dockerfile {
		do = append(do, sdk.WithBuilder(dockerfile.NewBuilder()))
	} else {
		do = append(do, sdk.WithBuilder(pack.NewBuilder()))
	}
	return do, nil
}

func (d deployConfig) deployClientOptions() ([]sdk.Option, error) {
	i, err := d.imageClientOptions()
	if err != nil {
		return i, err
	}

	return i, nil
}

func imageHubCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	iArgs := &imageArgs{}
	hc := &cobra.Command{
		Use:   "hub",
		Short: "Build and Push to images",
		Long:  "The hub subcommand used to build and push images",
		Example: `  # Build an image using a Dockerfile.
  dubboctl image hub -f Dockerfile

  # Build an image using a builder.
  dubboctl image hub -b
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if !iArgs.dockerfile && !iArgs.builder {
				return fmt.Errorf("at least one of the -b or -f flags must be set")
			}

			if cmd.Flags().Changed("file") {
				if len(args) != 1 {
					return fmt.Errorf("you must provide exactly one argument when using the -f flag: the path to the Dockerfile")
				}

				df := args[0]
				if !strings.HasSuffix(df, "Dockerfile") {
					return fmt.Errorf("the provided file must be a Dockerfile when using the -f flag")
				}
			}
			return nil
		},
		PreRunE: bindEnv("file", "builder"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHub(cmd, args, clientFactory)
		},
	}
	addHubFlags(hc, iArgs)
	return hc
}

func (hc *hubConfig) checkHubConfig(dc *dubbo.DubboConfig) {
	if hc.Path == "" {
		root, err := os.Getwd()
		if err != nil {
			return
		}
		dc.Root = root
	} else {
		dc.Root = hc.Path
	}
	if hc.BuilderImage != "" {
		dc.Build.BuilderImages["pack"] = hc.BuilderImage
	}
	if hc.Image != "" {
		dc.Image = hc.Image
	}
}

func runHub(cmd *cobra.Command, args []string, clientFactory ClientFactory) error {
	if err := util.GetCreatePath(); err != nil {
		return err
	}
	hubCfg := newHubConfig(cmd)
	filePath, err := dubbo.NewDubboConfig(hubCfg.Path)
	if err != nil {
		return err
	}

	hubCfg, err = hubCfg.hubPrompt(filePath)
	if err != nil {
		return err
	}

	if !filePath.Initialized() {
		return util.NewErrNotInitialized(filePath.Root)
	}

	hubCfg.checkHubConfig(filePath)

	clientOptions, err := hubCfg.imageClientOptions()
	if err != nil {
		return err
	}

	client, done := clientFactory(clientOptions...)
	defer done()

	filePath.Built()

	if filePath, err = client.Build(cmd.Context(), filePath); err != nil {
		return err
	}

	if filePath, err = client.Push(cmd.Context(), filePath); err != nil {
		return err
	}

	err = filePath.WriteFile()
	if err != nil {
		return err
	}

	return nil
}

func imageDeployCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	iArgs := &imageArgs{}
	hc := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy to cluster",
		Long:  "The deploy subcommand used to deploy to cluster",
		Example: `  # Deploy the application to the cluster.
  dubboctl image deploy

  # Delete the deployed application.
  dubboctl image deploy -d
`,
		PreRunE: bindEnv("output", "delete"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(cmd, args, clientFactory)
		},
	}
	addDeployFlags(hc, iArgs)
	return hc
}

func (d deployConfig) checkDeployConfig(dc *dubbo.DubboConfig) {
	d.checkHubConfig(dc)
	if d.Output != "" {
		dc.Deploy.Output = d.Output
	}
	if d.Namespace != "" {
		dc.Deploy.Namespace = d.Namespace
	}
	if d.Port != 0 {
		dc.Deploy.Port = d.Port
	}
}

func runDeploy(cmd *cobra.Command, args []string, clientFactory ClientFactory) error {
	if err := util.GetCreatePath(); err != nil {
		return err
	}
	deployCfg := newDeployConfig(cmd)

	filePath, err := dubbo.NewDubboConfig(deployCfg.Path)
	if err != nil {
		return err
	}

	deployCfg, err = deployCfg.deployPrompt(filePath)
	if err != nil {
		return err
	}

	deployCfg.checkDeployConfig(filePath)

	clientOptions, err := deployCfg.deployClientOptions()
	if err != nil {
		return err
	}

	client, done := clientFactory(clientOptions...)
	defer done()
	if filePath, err = client.Deploy(cmd.Context(), filePath); err != nil {
		return err
	}

	if !deployCfg.Destroy {
		if err := apply(cmd, filePath); err != nil {
			return err
		}
	}
	if deployCfg.Destroy {
		if err := remove(cmd, filePath); err != nil {
			return err
		}
	}

	err = filePath.WriteFile()
	if err != nil {
		return err
	}

	return nil
}

func apply(cmd *cobra.Command, dc *dubbo.DubboConfig) error {
	file := filepath.Join(dc.Root, dc.Deploy.Output)
	ec := exec.CommandContext(cmd.Context(), "kubectl", "apply", "-f", file)
	ec.Stdout, ec.Stderr = os.Stdout, os.Stderr
	if err := ec.Run(); err != nil {
		return err
	}
	return nil
}

func remove(cmd *cobra.Command, dc *dubbo.DubboConfig) error {
	file := filepath.Join(dc.Root, dc.Deploy.Output)
	ec := exec.CommandContext(cmd.Context(), "kubectl", "delete", "-f", file)
	ec.Stdout, ec.Stderr = os.Stdout, os.Stderr
	if err := ec.Run(); err != nil {
		return err
	}
	return nil
}

func (hc *hubConfig) hubPrompt(dc *dubbo.DubboConfig) (*hubConfig, error) {
	var err error
	if !util.InteractiveTerminal() {
		return hc, nil
	}

	if hc.Image == "" && dc.Image == "" {
		qs := []*survey.Question{
			{
				Name:     "image",
				Validate: survey.Required,
				Prompt: &survey.Input{
					Message: "Please enter the image tag ([REGISTRY]/[USERNAME]/[IMAGENAME]:tag)\n  Image: ",
					Default: hc.Image,
				},
			},
		}
		if err = survey.Ask(qs, hc); err != nil {
			return hc, err
		}
	}
	return hc, err
}

func (d *deployConfig) deployPrompt(dc2 *dubbo.DubboConfig) (*deployConfig, error) {
	var err error
	if !util.InteractiveTerminal() {
		return d, nil
	}
	if dc2.Deploy.Namespace == "" {
		qs := []*survey.Question{
			{
				Name:     "namespace",
				Validate: survey.Required,
				Prompt: &survey.Input{
					Message: "Namespace",
				},
			},
		}
		if err = survey.Ask(qs, d); err != nil {
			return d, err
		}
	}

	buildconfig, err := d.hubConfig.hubPrompt(dc2)
	if err != nil {
		return d, err
	}

	d.hubConfig = buildconfig

	if dc2.Deploy.Port == 0 && d.Port == 0 {
		qs := []*survey.Question{
			{
				Name:     "port",
				Validate: survey.Required,
				Prompt: &survey.Input{
					Message: "Port",
				},
			},
		}
		if err = survey.Ask(qs, d); err != nil {
			return d, err
		}
	}

	return d, err
}
