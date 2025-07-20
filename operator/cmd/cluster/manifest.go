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

package cluster

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/apache/dubbo-admin/dubboctl/pkg/cli"
	"github.com/apache/dubbo-admin/operator/pkg/manifest"
	"github.com/apache/dubbo-admin/operator/pkg/render"
	"github.com/apache/dubbo-admin/operator/pkg/util/clog"
	"github.com/apache/dubbo-admin/pkg/common/util/slices"
	"github.com/apache/dubbo-admin/pkg/kube"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

type manifestGenerateArgs struct {
	// filenames is an array of paths to input DubboOperator CR files.
	// filenames []string
	// sets is a string with the format "path=value".
	sets []string
}

func (a *manifestGenerateArgs) String() string {
	var b strings.Builder
	// b.WriteString("filenames:   " + fmt.Sprint(a.filenames) + "\n")
	b.WriteString("sets:           " + fmt.Sprint(a.sets) + "\n")
	return b.String()
}

func addManifestGenerateFlags(cmd *cobra.Command, args *manifestGenerateArgs) {
	// cmd.PersistentFlags().StringSliceVarP(&args.filenames, "filenames", "f", nil, ``)
	cmd.PersistentFlags().StringArrayVarP(&args.sets, "set", "s", nil, `Override dubboOperator values, such as selecting profiles, etc.`)
}

func ManifestCmd(ctx cli.Context) *cobra.Command {
	rootArgs := &RootArgs{}
	mgcArgs := &manifestGenerateArgs{}
	mgc := manifestGenerateCmd(ctx, rootArgs, mgcArgs)
	mc := &cobra.Command{
		Use:   "manifest",
		Short: "dubbo manifest related commands",
		Long:  "The manifest command will generates dubbo manifests.",
	}
	AddFlags(mc, rootArgs)
	AddFlags(mgc, rootArgs)
	addManifestGenerateFlags(mgc, mgcArgs)
	mc.AddCommand(mgc)
	return mc
}

var kubeClientFunc func() (kube.CLIClient, error)

func manifestGenerateCmd(ctx cli.Context, _ *RootArgs, mgArgs *manifestGenerateArgs) *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generates an Dubbo install manifest",
		Long:  "The generate subcommand generates an Dubbo install manifest and outputs to the console by default.",
		Example: `  # Generate a default Dubbo installation
  dubboctl manifest generate
  
  # Generate the demo profile
  dubboctl manifest generate --set profile=demo
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if kubeClientFunc == nil {
				kubeClientFunc = ctx.CLIClient
			}
			var kubeClient kube.CLIClient
			kc, err := kubeClientFunc()
			if err != nil {
				return err
			}
			kubeClient = kc

			cl := clog.NewConsoleLogger(cmd.OutOrStdout(), cmd.ErrOrStderr())
			return manifestGenerate(kubeClient, mgArgs, cl)
		},
	}
}

const (
	YAMLSeparator = "\n---\n"
)

func manifestGenerate(kc kube.CLIClient, mgArgs *manifestGenerateArgs, cl clog.Logger) error {
	setFlags := applyFlagAliases(mgArgs.sets)
	manifests, _, err := render.GenerateManifest(nil, setFlags, cl, kc)
	if err != nil {
		return err
	}
	for _, smf := range sortManifests(manifests) {
		cl.Print(smf + YAMLSeparator)
	}
	return nil
}

func sortManifests(raw []manifest.ManifestSet) []string {
	all := []manifest.Manifest{}
	for _, m := range raw {
		all = append(all, m.Manifests...)
	}
	slices.SortStableFunc(all, func(a, b manifest.Manifest) int {
		if r := cmp.Compare(objectKindOrder(a), objectKindOrder(b)); r != 0 {
			return r
		}
		if r := cmp.Compare(a.GroupVersionKind().Group, b.GroupVersionKind().Group); r != 0 {
			return r
		}
		if r := cmp.Compare(a.GroupVersionKind().Kind, b.GroupVersionKind().Kind); r != 0 {
			return r
		}
		return cmp.Compare(a.GetName(), b.GetName())
	})
	return slices.Map(all, func(e manifest.Manifest) string {
		res, _ := yaml.Marshal(e.Object)
		return string(res)
	})
}

func objectKindOrder(m manifest.Manifest) int {
	o := m.Unstructured
	gk := o.GroupVersionKind().Group + "/" + o.GroupVersionKind().Kind
	switch {
	// Create CRDs asap - both because they are slow and because we will likely create instances of them soon
	case gk == "apiextensions.k8s.io/CustomResourceDefinition":
		return -1000

		// We need to create ServiceAccounts, Roles before we bind them with a RoleBinding
	case gk == "/ServiceAccount" || gk == "rbac.authorization.k8s.io/ClusterRole":
		return 1
	case gk == "rbac.authorization.k8s.io/ClusterRoleBinding":
		return 2

		// Pods might need configmap or secrets - avoid backoff by creating them first
	case gk == "/ConfigMap" || gk == "/Secrets":
		return 100

		// Create the pods after we've created other things they might be waiting for
	case gk == "extensions/Deployment" || gk == "apps/Deployment":
		return 1000

		// Autoscalers typically act on a deployment
	case gk == "autoscaling/HorizontalPodAutoscaler":
		return 1001

		// Create services late - after pods have been started
	case gk == "/Service":
		return 10000

	default:
		return 1000
	}
}
