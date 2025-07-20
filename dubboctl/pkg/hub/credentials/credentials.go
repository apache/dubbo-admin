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

package credentials

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/pusher"
	dockerConfig "github.com/containers/image/v5/pkg/docker/config"
	containersTypes "github.com/containers/image/v5/types"
	"github.com/docker/docker-credential-helpers/client"
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	errUnauthorized                 = errors.New("bad credentials")
	errCredentialsNotFound          = errors.New("credentials not found")
	errNoCredentialHelperConfigured = errors.New("no credential helper configure")
)

type keyChain struct {
	user string
	pwd  string
}

type VerifyCredentialsCallback func(ctx context.Context, image string, credentials pusher.Credentials) error

type CredentialsCallback func(registry string) (pusher.Credentials, error)

type ChooseCredentialHelperCallback func(available []string) (string, error)

type credentialsProvider struct {
	promptForCredentials     CredentialsCallback
	verifyCredentials        VerifyCredentialsCallback
	promptForCredentialStore ChooseCredentialHelperCallback
	credentialLoaders        []CredentialsCallback
	authFilePath             string
	transport                http.RoundTripper
}

func NewCredentialsProvider(configPath string, opts ...Opt) pusher.CredentialsProvider {
	var c credentialsProvider

	for _, o := range opts {
		o(&c)
	}

	if c.transport == nil {
		c.transport = http.DefaultTransport
	}

	if c.verifyCredentials == nil {
		c.verifyCredentials = func(ctx context.Context, registry string, credentials pusher.Credentials) error {
			return checkAuth(ctx, registry, credentials, c.transport)
		}
	}

	if c.promptForCredentialStore == nil {
		c.promptForCredentialStore = func(available []string) (string, error) {
			return "", nil
		}
	}

	c.authFilePath = filepath.Join(configPath, "auth.json")
	sys := &containersTypes.SystemContext{
		AuthFilePath: c.authFilePath,
	}

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	dockerConfigPath := filepath.Join(home, ".docker", "config.json")

	defaultCredentialLoaders := []CredentialsCallback{
		func(registry string) (pusher.Credentials, error) {
			return getCredentialsByCredentialHelper(c.authFilePath, registry)
		},
		func(registry string) (pusher.Credentials, error) {
			return getCredentialsByCredentialHelper(dockerConfigPath, registry)
		},
		func(registry string) (pusher.Credentials, error) {
			creds, err := dockerConfig.GetCredentials(sys, registry)
			if err != nil {
				return pusher.Credentials{}, err
			}
			return pusher.Credentials{
				Username: creds.Username,
				Password: creds.Password,
			}, nil
		},
		func(registry string) (pusher.Credentials, error) {
			return pusher.Credentials{}, nil
		},
	}

	c.credentialLoaders = append(c.credentialLoaders, defaultCredentialLoaders...)

	return c.getCredentials
}

func (c *credentialsProvider) getCredentials(ctx context.Context, image string) (pusher.Credentials, error) {
	var err error
	result := pusher.Credentials{}

	ref, err := name.ParseReference(image)
	if err != nil {
		return pusher.Credentials{}, fmt.Errorf("cannot parse the image reference: %w", err)
	}

	registry := ref.Context().RegistryStr()

	for _, load := range c.credentialLoaders {

		result, err = load(registry)

		if err != nil {
			if errors.Is(err, errCredentialsNotFound) {
				continue
			}
			return pusher.Credentials{}, err
		}

		err = c.verifyCredentials(ctx, image, result)
		if err == nil {
			return result, nil
		} else {
			if !errors.Is(err, errUnauthorized) {
				return pusher.Credentials{}, err
			}
		}

	}

	if c.promptForCredentials == nil {
		return pusher.Credentials{}, errCredentialsNotFound
	}

	for {
		result, err = c.promptForCredentials(registry)
		if err != nil {
			return pusher.Credentials{}, err
		}

		err = c.verifyCredentials(ctx, image, result)
		if err == nil {
			err = setCredentialsByCredentialHelper(c.authFilePath, registry, result.Username, result.Password)
			if err != nil {

				if strings.Contains(err.Error(), "not implemented") {
					fmt.Fprintf(os.Stderr, "the cred-helper does not support write operation (consider changing the cred-helper it in auth.json)\n")
					return pusher.Credentials{}, nil
				}

				if !errors.Is(err, errNoCredentialHelperConfigured) {
					return pusher.Credentials{}, err
				}
				helpers := listCredentialHelpers()
				helper, err := c.promptForCredentialStore(helpers)
				if err != nil {
					return pusher.Credentials{}, err
				}
				helper = strings.TrimPrefix(helper, "docker-credential-")
				err = setCredentialHelperToConfig(c.authFilePath, helper)
				if err != nil {
					return pusher.Credentials{}, fmt.Errorf("faild to set the helper to the config: %w", err)
				}
				err = setCredentialsByCredentialHelper(c.authFilePath, registry, result.Username, result.Password)
				if err != nil {

					if strings.Contains(err.Error(), "not implemented") {
						fmt.Fprintf(os.Stderr, "the cred-helper does not support write operation (consider changing the cred-helper it in auth.json)\n")
						return pusher.Credentials{}, nil
					}

					if !errors.Is(err, errNoCredentialHelperConfigured) {
						return pusher.Credentials{}, err
					}
				}
			}
			return result, nil
		} else {
			if errors.Is(err, errUnauthorized) {
				continue
			}
			return pusher.Credentials{}, err
		}
	}
}

func (k keyChain) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	return &authn.Basic{
		Username: k.user,
		Password: k.pwd,
	}, nil
}

func checkAuth(ctx context.Context, image string, credentials pusher.Credentials, trans http.RoundTripper) error {
	ref, err := name.ParseReference(image)
	if err != nil {
		return fmt.Errorf("cannot parse image reference: %w", err)
	}

	kc := keyChain{
		user: credentials.Username,
		pwd:  credentials.Password,
	}

	err = remote.CheckPushPermission(ref, kc, trans)
	if err != nil {
		var transportErr *transport.Error
		if errors.As(err, &transportErr) && transportErr.StatusCode == 401 {
			return errUnauthorized
		}
		return err
	}

	return nil
}

func getCredentialHelperFromConfig(confFilePath string) (string, error) {
	data, err := os.ReadFile(confFilePath)
	if err != nil {
		return "", err
	}

	conf := struct {
		Store string `json:"credsStore"`
	}{}

	err = json.Unmarshal(data, &conf)
	if err != nil {
		return "", err
	}

	return conf.Store, nil
}

func setCredentialHelperToConfig(confFilePath, helper string) error {
	var err error

	configData := make(map[string]interface{})

	if data, err := os.ReadFile(confFilePath); err == nil {
		err = json.Unmarshal(data, &configData)
		if err != nil {
			return err
		}
	}

	configData["credsStore"] = helper

	data, err := json.MarshalIndent(&configData, "", "    ")
	if err != nil {
		return err
	}

	err = os.WriteFile(confFilePath, data, 0o600)
	if err != nil {
		return err
	}

	return nil
}

func getCredentialsByCredentialHelper(confFilePath, registry string) (pusher.Credentials, error) {
	result := pusher.Credentials{}

	helper, err := getCredentialHelperFromConfig(confFilePath)
	if err != nil && !os.IsNotExist(err) {
		return result, fmt.Errorf("failed to get helper from config: %w", err)
	}
	if helper == "" {
		return result, errCredentialsNotFound
	}

	helperName := fmt.Sprintf("docker-credential-%s", helper)
	p := client.NewShellProgramFunc(helperName)

	credentialsMap, err := client.List(p)
	if err != nil {
		return result, fmt.Errorf("failed to list credentials: %w", err)
	}

	for serverUrl := range credentialsMap {
		if RegistryEquals(serverUrl, registry) {
			creds, err := client.Get(p, serverUrl)
			if err != nil {
				return result, fmt.Errorf("failed to get credentials: %w", err)
			}
			result.Username = creds.Username
			result.Password = creds.Secret
			return result, nil
		}
	}

	return result, fmt.Errorf("failed to get credentials from helper specified in ~/.docker/config.json: %w", errCredentialsNotFound)
}

func setCredentialsByCredentialHelper(confFilePath, registry, username, secret string) error {
	helper, err := getCredentialHelperFromConfig(confFilePath)

	if helper == "" || os.IsNotExist(err) {
		return errNoCredentialHelperConfigured
	}
	if err != nil {
		return fmt.Errorf("failed to get helper from config: %w", err)
	}

	helperName := fmt.Sprintf("docker-credential-%s", helper)
	p := client.NewShellProgramFunc(helperName)

	return client.Store(p, &credentials.Credentials{ServerURL: registry, Username: username, Secret: secret})
}

func listCredentialHelpers() []string {
	path := os.Getenv("PATH")
	paths := strings.Split(path, string(os.PathListSeparator))

	helpers := make(map[string]bool)
	for _, p := range paths {
		fss, err := os.ReadDir(p)
		if err != nil {
			continue
		}
		for _, fi := range fss {
			if fi.IsDir() {
				continue
			}
			if !strings.HasPrefix(fi.Name(), "docker-credential-") {
				continue
			}
			if runtime.GOOS == "windows" {
				ext := filepath.Ext(fi.Name())
				if ext != ".exe" && ext != ".bat" {
					continue
				}
			}
			helpers[fi.Name()] = true
		}
	}
	result := make([]string, 0, len(helpers))
	for h := range helpers {
		result = append(result, h)
	}
	return result
}

func hostPort(registry string) (host string, port string) {
	if !strings.Contains(registry, "://") {
		h, p, err := net.SplitHostPort(registry)

		if err == nil {
			host, port = h, p
			return
		}
		registry = "https://" + registry
	}

	u, err := url.Parse(registry)
	if err != nil {
		panic(err)
	}
	host = u.Hostname()
	port = u.Port()
	return
}

func RegistryEquals(regA, regB string) bool {
	h1, p1 := hostPort(regA)
	h2, p2 := hostPort(regB)

	isStdPort := func(p string) bool { return p == "443" || p == "80" }

	portEq := p1 == p2 ||
		(p1 == "" && isStdPort(p2)) ||
		(isStdPort(p1) && p2 == "")

	if h1 == h2 && portEq {
		return true
	}

	if strings.HasSuffix(h1, "docker.io") &&
		strings.HasSuffix(h2, "docker.io") {
		return true
	}

	return false
}

type Opt func(opts *credentialsProvider)

func WithPromptForCredentials(cbk CredentialsCallback) Opt {
	return func(opts *credentialsProvider) {
		opts.promptForCredentials = cbk
	}
}

func WithPromptForCredentialStore(cbk ChooseCredentialHelperCallback) Opt {
	return func(opts *credentialsProvider) {
		opts.promptForCredentialStore = cbk
	}
}

func WithTransport(transport http.RoundTripper) Opt {
	return func(opts *credentialsProvider) {
		opts.transport = transport
	}
}
