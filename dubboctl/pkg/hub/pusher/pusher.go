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

package pusher

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"golang.org/x/term"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
)

type Opt func(*Pusher)

type Credentials struct {
	Username string
	Password string
}

type CredentialsProvider func(ctx context.Context, image string) (Credentials, error)

type DockerClientFactory func() (DockerClient, error)

type Pusher struct {
	credentialsProvider CredentialsProvider
	transport           http.RoundTripper
	dockerClientFactory DockerClientFactory
}

type authConfig struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func NewPusher(opts ...Opt) *Pusher {
	result := &Pusher{
		credentialsProvider: EmptyCredentialsProvider,
		transport:           http.DefaultTransport,
		dockerClientFactory: func() (DockerClient, error) {
			c, _, err := hub.NewClient(client.DefaultDockerHost)
			return c, err
		},
	}
	for _, opt := range opts {
		opt(result)
	}

	return result
}

func (p *Pusher) Push(ctx context.Context, dc *dubbo.DubboConfig) (digest string, err error) {
	var output io.Writer

	output = os.Stderr

	if dc.Image == "" {
		return "", errors.New("Function has no associated image.  Has it been built?")
	}

	registry, err := getRegistry(dc.Image)
	if err != nil {
		return "", err
	}

	credentials, err := p.credentialsProvider(ctx, dc.Image)
	if err != nil {
		return "", fmt.Errorf("failed to get credentials: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Pushing function image to the registry %q using the %q user credentials\n", registry, credentials.Username)

	if _, err = net.DefaultResolver.LookupHost(ctx, registry); err == nil {
		return p.daemonPush(ctx, dc, credentials, output)
	}

	return p.push(ctx, dc, credentials, output)
}

func getRegistry(img string) (string, error) {
	ref, err := name.ParseReference(img, name.WeakValidation)
	if err != nil {
		return "", err
	}
	registry := ref.Context().RegistryStr()
	return registry, nil
}

func (p *Pusher) daemonPush(ctx context.Context, dc *dubbo.DubboConfig, credentials Credentials, output io.Writer) (digest string, err error) {
	cli, err := p.dockerClientFactory()
	if err != nil {
		return "", fmt.Errorf("failed to create docker api client: %w", err)
	}
	defer cli.Close()

	ac := authConfig{
		Username: credentials.Username,
		Password: credentials.Password,
	}

	b, err := json.Marshal(&ac)
	if err != nil {
		return "", err
	}

	opts := types.ImagePushOptions{RegistryAuth: base64.StdEncoding.EncodeToString(b)}

	r, err := cli.ImagePush(ctx, dc.Image, opts)
	if err != nil {
		return "", fmt.Errorf("failed to push the image: %w", err)
	}
	defer r.Close()

	var outBuff bytes.Buffer
	output = io.MultiWriter(&outBuff, output)

	var isTerminal bool
	var fd uintptr
	if outF, ok := output.(*os.File); ok {
		fd = outF.Fd()
		isTerminal = term.IsTerminal(int(outF.Fd()))
	}

	err = jsonmessage.DisplayJSONMessagesStream(r, output, fd, isTerminal, nil)
	if err != nil {
		return "", err
	}

	return parseDigest(outBuff.String()), nil
}

var digestRE = regexp.MustCompile(`digest:\s+(sha256:\w{64})`)

func parseDigest(output string) string {
	match := digestRE.FindStringSubmatch(output)
	if len(match) >= 2 {
		return match[1]
	}
	return ""
}

func (n *Pusher) push(ctx context.Context, dc *dubbo.DubboConfig, credentials Credentials, output io.Writer) (digest string, err error) {
	auth := &authn.Basic{
		Username: credentials.Username,
		Password: credentials.Password,
	}

	ref, err := name.ParseReference(dc.Image)
	if err != nil {
		return "", err
	}

	dockerClient, err := n.dockerClientFactory()
	if err != nil {
		return "", fmt.Errorf("failed to create docker api client: %w", err)
	}
	defer dockerClient.Close()

	img, err := daemon.Image(ref,
		daemon.WithContext(ctx),
		daemon.WithClient(dockerClient))
	if err != nil {
		return "", err
	}

	progressChannel := make(chan v1.Update, 1024)
	errChan := make(chan error)
	go func() {
		defer fmt.Fprint(output, "\n")

		for progress := range progressChannel {
			if progress.Error != nil {
				errChan <- progress.Error
				return
			}
			fmt.Fprintf(output, "\rprogress: %d%%", progress.Complete*100/progress.Total)
		}

		errChan <- nil
	}()

	err = remote.Write(ref, img,
		remote.WithAuth(auth),
		remote.WithProgress(progressChannel),
		remote.WithTransport(n.transport),
		remote.WithJobs(1),
		remote.WithContext(ctx))
	if err != nil {
		return "", err
	}
	err = <-errChan
	if err != nil {
		return "", err
	}

	hash, err := img.Digest()
	if err != nil {
		return "", err
	}

	return hash.String(), nil
}

type DockerClient interface {
	daemon.Client
	ImagePush(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error)
	Close() error
}

func WithTransport(transport http.RoundTripper) Opt {
	return func(pusher *Pusher) {
		pusher.transport = transport
	}
}

func WithCredentialsProvider(cp CredentialsProvider) Opt {
	return func(p *Pusher) {
		p.credentialsProvider = cp
	}
}

func EmptyCredentialsProvider(ctx context.Context, registry string) (Credentials, error) {
	return Credentials{}, nil
}
