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
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"github.com/apache/dubbo-kubernetes/operator/cmd/cluster"
	"github.com/spf13/cobra"
)

type repoArgs struct{}

func addRepoFlags(cmd *cobra.Command, rArgs *repoArgs) {}

func RepoCmd(_ cli.Context, cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	rootArgs := &cluster.RootArgs{}
	rArgs := &repoArgs{}
	ad := addCmd(cmd, clientFactory)
	li := listCmd(cmd, clientFactory)
	re := removeCmd(cmd, clientFactory)
	rc := &cobra.Command{
		Use:   "repo",
		Short: "Manage exist Dubbo sdk module libraries",
		Long:  "The repo command Manage existing Dubbo SDK module libraries",
		Example: `  # Add a new template library.
  dubboctl repo add [<name>] [<url>]
	
  # View the list of template library.
  dubboctl repo list
	
  # Remove an existing template library.
  dubboctl repo remove [<name>]
`,
	}

	cluster.AddFlags(rc, rootArgs)
	addRepoFlags(rc, rArgs)
	rc.AddCommand(ad)
	rc.AddCommand(li)
	rc.AddCommand(re)
	return rc
}

func addCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	ac := &cobra.Command{
		Use:   "add [<name>] [<url>]",
		Short: "Add a new template library.",
		Long:  "The add subcommand is used to add a new template library.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(cmd, args, clientFactory)
		},
	}
	return ac
}

func runAdd(cmd *cobra.Command, args []string, clientFactory ClientFactory) (err error) {
	// Adding a repository requires there be a config path structure on disk
	if err = util.GetCreatePath(); err != nil {
		return
	}
	// Create a client instance which utilizes the given repositories path.
	// Note that this MAY not be in the config structure if the environment
	// variable to override said path was provided explicitly.
	// be created in XDG_CONFIG_HOME/dubbo even if the repo path environment
	// was set to some other location on disk.
	client, done := clientFactory()
	defer done()

	// Preconditions
	// If not confirming/prompting, assert the args were both provided.
	if len(args) != 2 {
		return fmt.Errorf("Usage: dubboctl repo add [<name>] [<url>]")
	}

	// Extract Params
	// Populate a struct with the arguments (if provided).
	p := struct {
		name string
		url  string
	}{}
	if len(args) > 0 {
		p.name = args[0]
	}
	if len(args) > 1 {
		p.url = args[1]
	}

	var n string
	if n, err = client.Repositories().Add(p.name, p.url); err != nil {
		return
	}

	fmt.Printf("%s Repositories added.\n", n)
	return
}

func listCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	lc := &cobra.Command{
		Use:     "list",
		Short:   "View the list of template library.",
		Long:    "The list subcommand is used to view the repositories that have been added.",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, args, clientFactory)
		},
	}
	return lc
}

func runList(cmd *cobra.Command, args []string, clientFactory ClientFactory) (err error) {
	client, done := clientFactory()
	defer done()

	list, err := client.Repositories().All()
	if err != nil {
		return
	}

	for _, l := range list {
		fmt.Println(l.Name + "\t" + l.URL())
	}
	return
}

func removeCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	rc := &cobra.Command{
		Use:     "remove [<name>]",
		Short:   "Remove an existing template library.",
		Long:    "The delete subcommand is used to delete a template from an existing repository.",
		Aliases: []string{"delete"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd, args, clientFactory)
		},
	}
	return rc
}

func runRemove(cmd *cobra.Command, args []string, clientFactory ClientFactory) (err error) {
	client, done := clientFactory()
	defer done()

	p := struct {
		name string
		sure bool
	}{}
	if len(args) > 0 {
		p.name = args[0]
	}
	p.sure = true

	if err = client.Repositories().Remove(p.name); err != nil {
		return
	}

	fmt.Printf("%s Repositories removed.\n", p.name)
	return
}
