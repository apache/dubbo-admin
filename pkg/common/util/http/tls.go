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

package http

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

func ConfigureMTLS(httpClient *http.Client, caCert string, clientCert string, clientKey string) error {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	if caCert == "" {
		transport.TLSClientConfig.InsecureSkipVerify = true
	} else {
		certBytes, err := os.ReadFile(caCert)
		if err != nil {
			return errors.Wrap(err, "could not read CA cert")
		}
		certPool := x509.NewCertPool()
		if ok := certPool.AppendCertsFromPEM(certBytes); !ok {
			return errors.New("could not add certificate")
		}
		transport.TLSClientConfig.RootCAs = certPool
	}

	if clientKey != "" && clientCert != "" {
		cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
		if err != nil {
			return errors.Wrap(err, "could not create key pair from client cert and client key")
		}
		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}

	httpClient.Transport = transport
	return nil
}
