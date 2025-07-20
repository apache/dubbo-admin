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

package sdk

import (
	"bufio"
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	templates        *Templates
	repositories     *Repositories
	repositoriesPath string
	builder          Builder
	pusher           Pusher
	deployer         Deployer
}

type Builder interface {
	Build(context.Context, *dubbo.DubboConfig) error
}

type Pusher interface {
	Push(ctx context.Context, dc *dubbo.DubboConfig) (string, error)
}

type DeployOption func(f *DeployParams)

type Deployer interface {
	Deploy(context.Context, *dubbo.DubboConfig, ...DeployOption) (DeploymentResult, error)
}

type DeploymentResult struct {
	Status    Status
	Namespace string
}

type Status int

const (
	Failed Status = iota
	Deployed
)

type DeployParams struct {
	skipBuiltCheck bool
}
type Option func(client *Client)

func New(options ...Option) *Client {
	c := &Client{}
	for _, o := range options {
		o(c)
	}
	c.repositories = newRepositories(c)
	c.templates = newTemplates(c)

	return c
}

func (c *Client) Templates() *Templates {
	return c.templates
}

func (c *Client) Runtimes() ([]string, error) {
	runtimes := util.NewSortedSet()
	repos, err := c.Repositories().All()
	if err != nil {
		return []string{}, err
	}
	for _, repo := range repos {
		for _, runtime := range repo.Runtimes {
			runtimes.Add(runtime.Name)
		}
	}
	return runtimes.Items(), nil
}

func (c *Client) Repositories() *Repositories {
	return c.repositories
}

func (c *Client) Initialize(dcfg *dubbo.DubboConfig, initialized bool, cmd *cobra.Command) (*dubbo.DubboConfig, error) {
	var err error
	oldRoot := dcfg.Root

	dcfg.Root, err = filepath.Abs(dcfg.Root)
	if err != nil {
		return dcfg, err
	}
	if err = os.MkdirAll(dcfg.Root, 0o755); err != nil {
		return dcfg, err
	}

	has, err := hasInitialized(dcfg.Root)
	if err != nil {
		return dcfg, err
	}
	if has {
		return dcfg, fmt.Errorf("%v already initialized", dcfg.Root)
	}

	if dcfg.Root == "" {
		if dcfg.Root, err = os.Getwd(); err != nil {
			return dcfg, err
		}
	}
	if dcfg.Name == "" {
		dcfg.Name = nameFromPath(dcfg.Root)
	}

	if !initialized {
		if err := assertEmptyRoot(dcfg.Root); err != nil {
			return dcfg, err
		}
	}
	// TODO remove initiallized
	f := dubbo.NewDubboConfigWithTemplate(dcfg, initialized)

	if err = runDataDir(f.Root); err != nil {
		return f, err
	}

	if !initialized {
		err = c.Templates().Write(f)
		if err != nil {
			return f, err
		}
	}

	f.Created = time.Now()
	err = f.WriteFile()
	if err != nil {
		return f, err
	}
	err = f.WriteDockerfile(cmd)
	if err != nil {
		return f, err
	}

	return dubbo.NewDubboConfig(oldRoot)
}

type BuildOptions struct{}

type BuildOption func(c *BuildOptions)

func (c *Client) Build(ctx context.Context, dc *dubbo.DubboConfig, options ...BuildOption) (*dubbo.DubboConfig, error) {
	fmt.Println("Starting to built the image...")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	bo := BuildOptions{}
	for _, o := range options {
		o(&bo)
	}

	if err := c.builder.Build(ctx, dc); err != nil {
		return dc, err
	}
	if err := dc.Stamp(); err != nil {
		return dc, err
	}
	fmt.Printf("Image built completed: %v\n", dc.Image)
	return dc, nil
}

func (c *Client) Push(ctx context.Context, dc *dubbo.DubboConfig) (*dubbo.DubboConfig, error) {
	var err error
	if !dc.Built() {
		return dc, errors.New("not built")
	}
	if dc.ImageDigest, err = c.pusher.Push(ctx, dc); err != nil {
		return dc, err
	}

	return dc, nil
}

func (c *Client) Deploy(ctx context.Context, dc *dubbo.DubboConfig, opts ...DeployOption) (*dubbo.DubboConfig, error) {
	deployParams := &DeployParams{skipBuiltCheck: false}
	for _, opt := range opts {
		opt(deployParams)
	}

	go func() {
		<-ctx.Done()
	}()

	if dc.Name == "" {
		return dc, errors.New("name required")
	}
	result, err := c.deployer.Deploy(ctx, dc)
	if err != nil {
		fmt.Printf("deploy error: %v\n", err)
		return dc, err
	}

	dc.Deploy.Namespace = result.Namespace

	if result.Status == Deployed {
		// TODO
	}

	return dc, nil
}

func hasInitialized(path string) (bool, error) {
	var err error
	filename := filepath.Join(path, dubbo.LogFile)

	if _, err = os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	bb, err := os.ReadFile(filename)
	if err != nil {
		return false, err
	}
	f := dubbo.DubboConfig{}
	if err = yaml.Unmarshal(bb, &f); err != nil {
		return false, err
	}

	return f.Initialized(), nil
}

func nameFromPath(path string) string {
	pathParts := strings.Split(strings.TrimRight(path, string(os.PathSeparator)), string(os.PathSeparator))
	return pathParts[len(pathParts)-1]
}

func assertEmptyRoot(path string) (err error) {
	files, err := contentiousFilesIn(path)
	if err != nil {
		return
	} else if len(files) > 0 {
		return fmt.Errorf("the chosen directory '%v' contains contentious files: %v.  Has the Service function already been created?  Try either using a different directory, deleting the function if it exists, or manually removing the files", path, files)
	}
	empty, err := isEffectivelyEmpty(path)
	if err != nil {
		return
	} else if !empty {
		err = errors.New("the directory must be empty of visible files and recognized config files before it can be initialized")
		return
	}
	return
}

func runDataDir(root string) error {
	if err := os.MkdirAll(filepath.Join(root, dubbo.DataDir), os.ModePerm); err != nil {
		return err
	}
	filePath := filepath.Join(root, ".gitignore")
	roFile, err := os.Open(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer roFile.Close()
	if !os.IsNotExist(err) {
		s := bufio.NewScanner(roFile)
		for s.Scan() {
			if strings.HasPrefix(s.Text(), "# /"+dubbo.DataDir) { // if it was commented
				return nil // user wants it
			}
			if strings.HasPrefix(s.Text(), "#/"+dubbo.DataDir) {
				return nil // user wants it
			}
			if strings.HasPrefix(s.Text(), "/"+dubbo.DataDir) { // if it is there
				return nil // we're done
			}
		}
	}
	roFile.Close()
	rwFile, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer rwFile.Close()
	if _, err = rwFile.WriteString(`
# Applications use the .dubbo directory for local runtime data which should
# generally not be tracked in source control. To instruct the system to track
# .dubbo in source control, comment the following line (prefix it with '# ').
/.dubbo
`); err != nil {
		return err
	}
	if err = rwFile.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: error when syncing .gitignore. %s\n", err)
	}
	return nil
}

var contentiousFiles = []string{
	dubbo.LogFile,
	".gitignore",
}

func contentiousFilesIn(dir string) (contentious []string, err error) {
	files, err := os.ReadDir(dir)
	for _, file := range files {
		for _, name := range contentiousFiles {
			if file.Name() == name {
				contentious = append(contentious, name)
			}
		}
	}
	return
}

func isEffectivelyEmpty(dir string) (bool, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), ".") {
			return false, nil
		}
	}
	return true, nil
}

func WithRepositoriesPath(path string) Option {
	return func(c *Client) {
		c.repositoriesPath = path
	}
}

func WithBuilder(b Builder) Option {
	return func(c *Client) {
		c.builder = b
	}
}

func WithPusher(pusher Pusher) Option {
	return func(c *Client) {
		c.pusher = pusher
	}
}

func WithDeployer(d Deployer) Option {
	return func(c *Client) {
		c.deployer = d
	}
}
