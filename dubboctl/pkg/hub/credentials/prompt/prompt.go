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

package prompt

import (
	"bufio"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/credentials"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/pusher"
	"golang.org/x/term"
	"io"
	"os"
	"strings"
)

func NewPromptForCredentials(in io.Reader, out, errOut io.Writer) func(registry string) (pusher.Credentials, error) {
	firstTime := true
	return func(registry string) (pusher.Credentials, error) {
		var result pusher.Credentials

		if firstTime {
			firstTime = false
			fmt.Fprintf(out, "Please provide credentials for image registry (%s).\n", registry)
		} else {
			fmt.Fprintln(out, "Incorrect credentials, please try again.")
		}

		qs := []*survey.Question{
			{
				Name: "username",
				Prompt: &survey.Input{
					Message: "Username:",
				},
				Validate: survey.Required,
			},
			{
				Name: "password",
				Prompt: &survey.Password{
					Message: "Password:",
				},
				Validate: survey.Required,
			},
		}

		var (
			fr terminal.FileReader
			ok bool
		)

		isTerm := false
		if fr, ok = in.(terminal.FileReader); ok {
			isTerm = term.IsTerminal(int(fr.Fd()))
		}

		if isTerm {
			err := survey.Ask(qs, &result, survey.WithStdio(fr, out.(terminal.FileWriter), errOut))
			if err != nil {
				return pusher.Credentials{}, err
			}
		} else {
			reader := bufio.NewReader(in)

			fmt.Fprint(out, "Username: ")
			u, err := reader.ReadString('\n')
			if err != nil {
				return pusher.Credentials{}, err
			}
			u = strings.Trim(u, "\r\n")

			fmt.Fprint(out, "Password: ")
			p, err := reader.ReadString('\n')
			if err != nil {
				return pusher.Credentials{}, err
			}
			p = strings.Trim(p, "\r\n")

			result = pusher.Credentials{Username: u, Password: p}
		}

		return result, nil
	}
}

func NewPromptForCredentialStore() credentials.ChooseCredentialHelperCallback {
	return func(availableHelpers []string) (string, error) {
		if len(availableHelpers) < 1 {
			fmt.Fprintf(os.Stderr, `Credentials will not be saved.
If you would like to save your credentials in the future,
you can install docker credential helper https://github.com/docker/docker-credential-helpers.
`)
			return "", nil
		}

		isTerm := term.IsTerminal(int(os.Stdin.Fd()))

		var resp string

		if isTerm {
			err := survey.AskOne(&survey.Select{
				Message: "Choose credentials helper",
				Options: append(availableHelpers, "None"),
			}, &resp, survey.WithValidator(survey.Required))
			if err != nil {
				return "", err
			}
			if resp == "None" {
				fmt.Fprintf(os.Stderr, "No helper selected. Credentials will not be saved.\n")
				return "", nil
			}
		} else {
			fmt.Fprintf(os.Stderr, "Available credential helpers:\n")
			for _, helper := range availableHelpers {
				fmt.Fprintf(os.Stderr, "%s\n", helper)
			}
			fmt.Fprintf(os.Stderr, "Choose credentials helper: ")

			reader := bufio.NewReader(os.Stdin)

			var err error
			resp, err = reader.ReadString('\n')
			if err != nil {
				return "", err
			}
			resp = strings.Trim(resp, "\r\n")
			if resp == "" {
				fmt.Fprintf(os.Stderr, "No helper selected. Credentials will not be saved.\n")
			}
		}

		return resp, nil
	}
}
