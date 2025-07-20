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
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
	"io"
	"os"
	"strings"
)

func readSecret(prompt string) (pw []byte, err error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		fmt.Fprint(os.Stderr, prompt)
		pw, err = term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		return
	}
	var b [1]byte
	for {
		n, err := os.Stdin.Read(b[:])
		if n > 0 && b[0] != '\r' {
			if b[0] == '\n' {
				return pw, nil
			}
			pw = append(pw, b[0])
			if len(pw) > 1024 {
				err = errors.New("password too long, 1024 byte limit")
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) && len(pw) > 0 {
				err = nil
			}
			return pw, err
		}
	}
}

func NewPasswordCbk() PasswordCallback {
	var pwdSet bool
	var pwd string
	return func() (string, error) {
		if pwdSet {
			return pwd, nil
		}

		p, err := readSecret("please enter password:")
		if err != nil {
			return "", err
		}
		pwdSet = true
		pwd = string(p)

		return pwd, err
	}
}

func NewPassPhraseCbk() PassPhraseCallback {
	var pwdSet bool
	var pwd string
	return func() (string, error) {
		if pwdSet {
			return pwd, nil
		}

		p, err := readSecret("please enter passphrase to private key:")
		if err != nil {
			return "", err
		}
		pwdSet = true
		pwd = string(p)

		return pwd, err
	}
}

func NewHostKeyCbk() HostKeyCallback {
	var trust []byte
	return func(hostPort string, pubKey ssh.PublicKey) error {
		if bytes.Equal(trust, pubKey.Marshal()) {
			return nil
		}
		msg := `The authenticity of host %s cannot be established.
%s key fingerprint is %s
Are you sure you want to continue connecting (yes/no)? `
		fmt.Fprintf(os.Stderr, msg, hostPort, pubKey.Type(), ssh.FingerprintSHA256(pubKey))
		reader := bufio.NewReader(os.Stdin)
		answer, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		answer = strings.TrimRight(answer, "\r\n")
		answer = strings.ToLower(answer)

		if answer == "yes" || answer == "y" {
			trust = pubKey.Marshal()
			fmt.Fprintf(os.Stderr, "To avoid this in future add following line into your ~/.ssh/known_hosts:\n%s %s %s\n",
				hostPort, pubKey.Type(), base64.StdEncoding.EncodeToString(trust))
			return nil
		}

		return errors.New("key rejected")
	}
}
