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

package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type jwksDocument struct {
	Keys []jwkDocument `json:"keys"`
}

type jwkDocument struct {
	KeyType   string `json:"kty"`
	Use       string `json:"use"`
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	Modulus   string `json:"n"`
	Exponent  string `json:"e"`
}

// JWKSClient fetches the public signing keys published by the Admin API.
type JWKSClient struct {
	url        string
	httpClient *http.Client
}

func NewJWKSClient(url string, httpClient *http.Client) *JWKSClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &JWKSClient{url: url, httpClient: httpClient}
}

func (c *JWKSClient) Fetch(ctx context.Context) (jwksDocument, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return jwksDocument{}, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return jwksDocument{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return jwksDocument{}, fmt.Errorf("JWKS endpoint returned HTTP %d", resp.StatusCode)
	}
	var document jwksDocument
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&document); err != nil {
		return jwksDocument{}, err
	}
	return document, nil
}
