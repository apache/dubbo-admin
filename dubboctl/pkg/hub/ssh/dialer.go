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

package ssh

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/pkg/homedir"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
	"net"
	nurl "net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrBadServerKeyMsg     = "server key for given host differs from key in known_host"
	ErrUnknownServerKeyMsg = "server key not found in known_hosts"
)

// I would expose those but since ssh pkg doesn't do correct error wrapping it would be entirely futile
var (
	errBadServerKey     = errors.New(ErrBadServerKeyMsg)
	errUnknownServerKey = errors.New(ErrUnknownServerKeyMsg)
)

type (
	PasswordCallback   func() (string, error)
	PassPhraseCallback func() (string, error)
	HostKeyCallback    func(hostPort string, pubKey ssh.PublicKey) error
)

type Config struct {
	Identity           string
	PassPhrase         string
	PasswordCallback   PasswordCallback
	PassPhraseCallback PassPhraseCallback
	HostKeyCallback    HostKeyCallback
}

type dialer struct {
	sshClient *ssh.Client
	network   string
	addr      string
}

func (d *dialer) DialContext(ctx context.Context, n, a string) (net.Conn, error) {
	conn, err := d.Dial(d.network, d.addr)
	if err != nil {
		return nil, err
	}
	go func() {
		if ctx != nil {
			<-ctx.Done()
			conn.Close()
		}
	}()
	return conn, nil
}

func (d *dialer) Dial(n, a string) (net.Conn, error) {
	return d.sshClient.Dial(d.network, d.addr)
}

func (d *dialer) Close() error {
	return d.sshClient.Close()
}

type ContextDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type DialContextFn = func(ctx context.Context, network, addr string) (net.Conn, error)

func NewDialContext(url *nurl.URL, config Config) (ContextDialer, string, error) {
	sshConfig, err := NewSSHClientConfig(url, config)
	if err != nil {
		return nil, "", err
	}

	port := url.Port()
	if port == "" {
		port = "22"
	}
	host := url.Hostname()

	sshClient, err := ssh.Dial("tcp", net.JoinHostPort(host, port), sshConfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to dial ssh: %w", err)
	}
	defer func() {
		if sshClient != nil {
			sshClient.Close()
		}
	}()

	var remoteDockerHost string
	if url.Path != "" {
		remoteDockerHost = fmt.Sprintf(`unix://%s`, url.Path)
	} else {
		remoteDockerHost, err = getRemoteDockerHost(sshClient)
		if err != nil {
			return nil, "", err
		}
	}

	network, addr, err := getNetworkAndAddress(remoteDockerHost)
	if err != nil {
		return nil, "", err
	}

	if network == "npipe" {
		// ssh tunneling doesn't support tunneling of Windows' named pipes
		dialContext, err := stdioDialContext(url, sshClient, config.Identity)
		return contextDialerFn(dialContext), remoteDockerHost, err
	}

	d := dialer{sshClient: sshClient, addr: addr, network: network}
	// moving ownership of sshClient from this function to the returned structure
	sshClient = nil

	return &d, remoteDockerHost, nil
}

func getRemoteDockerHost(sshClient *ssh.Client) (remoteDockerHost string, err error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return
	}
	defer session.Close()

	out, err := session.CombinedOutput("set")
	if err != nil {
		return
	}

	remoteDockerHost = "unix:///var/run/docker.sock"
	isWin, err := isWindowsMachine(sshClient)
	if err != nil {
		return
	}

	if isWin {
		remoteDockerHost = "npipe:////./pipe/docker_engine"
	}

	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "DOCKER_HOST=") {
			parts := strings.SplitN(scanner.Text(), "=", 2)
			remoteDockerHost = strings.Trim(parts[1], `"'`)
			break
		}
	}

	return remoteDockerHost, err
}

func isWindowsMachine(sshClient *ssh.Client) (bool, error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return false, err
	}
	defer session.Close()

	out, err := session.CombinedOutput("systeminfo")
	if err == nil && strings.Contains(string(out), "Windows") {
		return true, nil
	}
	return false, nil
}

func stdioDialContext(url *nurl.URL, sshClient *ssh.Client, identity string) (DialContextFn, error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	out, err := session.CombinedOutput("docker system dial-stdio --help")
	if err != nil {
		return nil, fmt.Errorf("cannot use dial-stdio: %w (%q)", err, out)
	}

	var opts []string
	if identity != "" {
		opts = append(opts, "-i", identity)
	}

	connHelper, err := connhelper.GetConnectionHelperWithSSHOpts(url.String(), opts)
	if err != nil {
		return nil, err
	}

	return connHelper.Dialer, nil
}

func getNetworkAndAddress(remoteDockerHost string) (network string, addr string, err error) {
	remoteDockerHostURL, err := nurl.Parse(remoteDockerHost)
	if err != nil {
		return
	}
	switch remoteDockerHostURL.Scheme {
	case "unix", "npipe":
		addr = remoteDockerHostURL.Path
	case "fd":
		remoteDockerHostURL.Scheme = "tcp" // don't know why it works that way
		fallthrough
	case "tcp":
		addr = remoteDockerHostURL.Host
	default:
		return "", "", errors.New("scheme is not supported")
	}
	network = remoteDockerHostURL.Scheme

	return network, addr, err
}

var knownKeyNames = []string{"id_rsa", "id_dsa", "id_ecdsa", "id_ecdsa_sk", "id_ed25519", "id_ed25519_sk"}

func NewSSHClientConfig(url *nurl.URL, credentialsConfig Config) (*ssh.ClientConfig, error) {
	var (
		authMethods []ssh.AuthMethod
		signers     []ssh.Signer
		err         error
	)

	if pw, found := url.User.Password(); found {
		authMethods = append(authMethods, ssh.Password(pw))
	}

	// add signer from explicit identity parameter
	if credentialsConfig.Identity != "" {
		s, err := publicKey(credentialsConfig.Identity, []byte(credentialsConfig.Identity), credentialsConfig.PassPhraseCallback)
		if err != nil {
			return nil, fmt.Errorf("failed to parse identity file: %w", err)
		}
		signers = append(signers, s)
	}

	// add signers from ssh-agent
	if sock, found := os.LookupEnv("SSH_AUTH_SOCK"); found && sock != "" {
		var agentSigners []ssh.Signer
		var agentConn net.Conn
		agentConn, err = dialSSHAgentConnection(sock)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to ssh-agent's socket: %w", err)
		}
		agentSigners, err = agent.NewClient(agentConn).Signers()
		if err != nil {
			return nil, fmt.Errorf("failed to get signers from ssh-agent: %w", err)
		}
		signers = append(signers, agentSigners...)
	}

	// if there is no explicit identity file nor keys from ssh-agent then
	// add keys with standard name from ~/.ssh/
	if len(signers) == 0 {
		var defaultKeyPaths []string
		if home, err := os.UserHomeDir(); err == nil {
			for _, keyName := range knownKeyNames {
				p := filepath.Join(home, ".ssh", keyName)

				fi, err := os.Stat(p)
				if err != nil {
					continue
				}
				if fi.Mode().IsRegular() {
					defaultKeyPaths = append(defaultKeyPaths, p)
				}
			}
		}

		if len(defaultKeyPaths) == 1 {
			s, err := publicKey(defaultKeyPaths[0], []byte(credentialsConfig.PassPhrase), credentialsConfig.PassPhraseCallback)
			if err != nil {
				return nil, err
			}
			signers = append(signers, s)
		}
	}

	if len(signers) > 0 {
		dedup := make(map[string]ssh.Signer)
		// Dedup signers based on fingerprint, ssh-agent keys override explicit identity
		for _, s := range signers {
			fp := ssh.FingerprintSHA256(s.PublicKey())
			// if _, found := dedup[fp]; found {
			//	key updated
			// }
			dedup[fp] = s
		}

		var uniq []ssh.Signer
		for _, s := range dedup {
			uniq = append(uniq, s)
		}
		authMethods = append(authMethods, ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
			return uniq, nil
		}))
	}

	if len(authMethods) == 0 && credentialsConfig.PasswordCallback != nil {
		authMethods = append(authMethods, ssh.PasswordCallback(credentialsConfig.PasswordCallback))
	}

	const sshTimeout = 5
	clientConfig := &ssh.ClientConfig{
		User:            url.User.Username(),
		Auth:            authMethods,
		HostKeyCallback: createHostKeyCallback(credentialsConfig.HostKeyCallback),
		HostKeyAlgorithms: []string{
			ssh.KeyAlgoECDSA256,
			ssh.KeyAlgoECDSA384,
			ssh.KeyAlgoECDSA521,
			ssh.KeyAlgoED25519,
			ssh.KeyAlgoRSASHA256,
			ssh.KeyAlgoRSASHA512,
			ssh.KeyAlgoRSA,
			ssh.KeyAlgoDSA,
		},
		Timeout: sshTimeout * time.Second,
	}

	return clientConfig, nil
}

func publicKey(path string, passphrase []byte, passPhraseCallback PassPhraseCallback) (ssh.Signer, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		var missingPhraseError *ssh.PassphraseMissingError
		if ok := errors.As(err, &missingPhraseError); !ok {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		if len(passphrase) == 0 && passPhraseCallback != nil {
			b, err := passPhraseCallback()
			if err != nil {
				return nil, err
			}
			passphrase = []byte(b)
		}

		return ssh.ParsePrivateKeyWithPassphrase(key, passphrase)
	}

	return signer, nil
}

func createHostKeyCallback(hostKeyCallback HostKeyCallback) func(hostPort string, remote net.Addr, key ssh.PublicKey) error {
	return func(hostPort string, remote net.Addr, pubKey ssh.PublicKey) error {
		host, port := hostPort, "22"
		if _h, _p, err := net.SplitHostPort(host); err == nil {
			host, port = _h, _p
		}

		knownHosts := filepath.Join(homedir.Get(), ".ssh", "known_hosts")

		_, err := os.Stat(knownHosts)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			if hostKeyCallback != nil && hostKeyCallback(hostPort, pubKey) == nil {
				return nil
			}
			return errUnknownServerKey
		}

		f, err := os.Open(knownHosts)
		if err != nil {
			return fmt.Errorf("failed to open known_hosts: %w", err)
		}
		defer f.Close()

		hashhost := knownhosts.HashHostname(host)

		var errs []error
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			_, hostPorts, _key, _, _, err := ssh.ParseKnownHosts(scanner.Bytes())
			if err != nil {
				errs = append(errs, err)
				continue
			}

			for _, hp := range hostPorts {
				h, p := hp, "22"
				if _h, _p, err := net.SplitHostPort(hp); err == nil {
					h, p = _h, _p
				}

				if (h == host || h == hashhost) && port == p {
					if pubKey.Type() != _key.Type() {
						errs = append(errs, fmt.Errorf("missmatch in type of a key"))
						continue
					}
					if bytes.Equal(_key.Marshal(), pubKey.Marshal()) {
						return nil
					}

					return errBadServerKey
				}
			}
		}

		if hostKeyCallback != nil && hostKeyCallback(hostPort, pubKey) == nil {
			return nil
		}

		if len(errs) > 0 {
			return fmt.Errorf("server is not trusted (%v)", errs)
		}

		return errUnknownServerKey
	}
}

type contextDialerFn DialContextFn

func (n contextDialerFn) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return n(ctx, network, address)
}
