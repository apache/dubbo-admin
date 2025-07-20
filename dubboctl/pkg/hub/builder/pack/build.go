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

package pack

import (
	"bytes"
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/builder"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	pack "github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/project/types"
	"github.com/docker/docker/client"
	"github.com/heroku/color"
	"io"
	"runtime"
	"time"
)

const DefaultName = builder.Pack

var (
	DefaultGoBuilder   = "heroku/builder:24"
	DefaultJavaBuilder = "heroku/builder:24"
)

var (
	DefaultBuilderImages = map[string]string{
		"go":   DefaultGoBuilder,
		"java": DefaultJavaBuilder,
	}

	defaultBuildpacks = map[string][]string{}
)

type Builder struct {
	name          string
	outBuffer     bytes.Buffer
	logger        logging.Logger
	impl          Impl
	withTimestamp bool
}

type Impl interface {
	Build(context.Context, pack.BuildOptions) error
}

type Option func(*Builder)

func NewBuilder(options ...Option) *Builder {
	b := &Builder{name: DefaultName}
	for _, o := range options {
		o(b)
	}
	b.logger = logging.NewLogWithWriters(color.Stdout(), color.Stderr(), logging.WithVerbose())
	return b
}

func BuilderImage(dc *dubbo.DubboConfig, builderName string) (string, error) {
	return builder.Image(dc, builderName, DefaultBuilderImages)
}

func (b *Builder) Build(ctx context.Context, dc *dubbo.DubboConfig) (err error) {
	image, err := BuilderImage(dc, b.name)
	if err != nil {
		return
	}

	buildpacks := dc.Build.Buildpacks
	if len(buildpacks) == 0 {
		buildpacks = defaultBuildpacks[dc.Runtime]
	}

	opts := pack.BuildOptions{
		AppPath:    dc.Root,
		Image:      dc.Image,
		Builder:    image,
		Buildpacks: buildpacks,
		ProjectDescriptor: types.Descriptor{
			Build: types.Build{
				Exclude: []string{},
			},
		},
		ContainerConfig: struct {
			Network string
			Volumes []string
		}{Network: "", Volumes: nil},
	}
	if b.withTimestamp {
		now := time.Now()
		opts.CreationTime = &now
	}
	if runtime.GOOS == "linux" {
		opts.ContainerConfig.Network = "host"
	}

	impl := b.impl
	if impl == nil {
		var (
			cli        client.CommonAPIClient
			dockerHost string
		)

		cli, dockerHost, err = hub.NewClient(client.DefaultDockerHost)
		if err != nil {
			return fmt.Errorf("cannot create docker client: %w", err)
		}
		defer cli.Close()
		opts.DockerHost = dockerHost

		if impl, err = pack.NewClient(pack.WithLogger(b.logger), pack.WithDockerClient(cli)); err != nil {
			return fmt.Errorf("cannot create pack client: %w", err)
		}
	}

	if err = impl.Build(ctx, opts); err != nil {
		if ctx.Err() != nil {
			return
		} else {
			err = fmt.Errorf("failed to build the application: %w", err)
			fmt.Fprintln(color.Stderr(), "")
			_, _ = io.Copy(color.Stderr(), &b.outBuffer)
			fmt.Fprintln(color.Stderr(), "")
		}
	}
	return
}
