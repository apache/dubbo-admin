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

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type stubProvider struct {
	id            string
	displayName   string
	iconURL       string
	needsNonce    bool
	authenticated int
}

func (p *stubProvider) ID() string                   { return p.id }
func (p *stubProvider) DisplayName() string          { return p.displayName }
func (p *stubProvider) IconURL() string              { return p.iconURL }
func (p *stubProvider) NeedsNonce() bool             { return p.needsNonce }
func (p *stubProvider) PostLoginRedirectURL() string { return "https://admin.example/admin/" }
func (p *stubProvider) AuthorizationURL(transaction OAuthTransaction) string {
	return "https://provider.example/authorize?state=" + url.QueryEscape(transaction.State) + "&code_challenge=" + url.QueryEscape(PKCEChallenge(transaction.CodeVerifier)) + "&nonce=" + url.QueryEscape(transaction.Nonce)
}
func (p *stubProvider) Authenticate(_ context.Context, _, _, _ string) (Principal, error) {
	p.authenticated++
	return Principal{Subject: p.id + ":1"}, nil
}

func TestServicePublicProvidersAreSorted(t *testing.T) {
	service, err := newService([]Provider{
		&stubProvider{id: "zeta", displayName: "Zeta"},
		&stubProvider{id: "alpha", displayName: "Alpha", iconURL: "/admin/auth-providers/alpha.svg"},
	})
	if err != nil {
		t.Fatal(err)
	}
	providers := service.PublicProviders()
	if len(providers) != 2 || providers[0].ID != "alpha" || providers[1].ID != "zeta" {
		t.Fatalf("PublicProviders() = %+v", providers)
	}
	if providers[0].IconURL != "/admin/auth-providers/alpha.svg" {
		t.Fatalf("PublicProviders()[0].IconURL = %q", providers[0].IconURL)
	}
}

func TestServicePublicProvidersSerializesEmptyListAsArray(t *testing.T) {
	service, err := newService(nil)
	if err != nil {
		t.Fatal(err)
	}
	providers := service.PublicProviders()
	if providers == nil {
		t.Fatal("PublicProviders() returned nil, want an empty slice")
	}
	raw, err := json.Marshal(providers)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(raw) != "[]" {
		t.Fatalf("JSON = %s, want []", raw)
	}
}

func TestServiceBeginGeneratesStatePKCEAndOIDCNonce(t *testing.T) {
	service, _ := newService([]Provider{&stubProvider{id: "sso", needsNonce: true}})
	transaction, authorizationURL, err := service.Begin("sso")
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	if transaction.ProviderID != "sso" || len(transaction.State) < 43 || len(transaction.CodeVerifier) < 43 || len(transaction.Nonce) < 43 {
		t.Fatalf("transaction = %+v", transaction)
	}
	if strings.Contains(authorizationURL, transaction.CodeVerifier) {
		t.Fatalf("authorization URL leaked verifier: %s", authorizationURL)
	}
	parsed, _ := url.Parse(authorizationURL)
	if got := parsed.Query().Get("code_challenge"); got != PKCEChallenge(transaction.CodeVerifier) {
		t.Fatalf("code_challenge = %q", got)
	}
}

func TestServiceCompleteRejectsProviderAndStateBeforeAuthentication(t *testing.T) {
	provider := &stubProvider{id: "github"}
	service, _ := newService([]Provider{provider})
	transaction := OAuthTransaction{ProviderID: "github", State: "expected", CodeVerifier: "verifier"}

	if _, err := service.Complete(context.Background(), "other", "expected", "code", transaction); !errors.Is(err, ErrProviderMismatch) {
		t.Fatalf("provider mismatch error = %v", err)
	}
	if _, err := service.Complete(context.Background(), "github", "wrong", "code", transaction); !errors.Is(err, ErrStateMismatch) {
		t.Fatalf("state mismatch error = %v", err)
	}
	if provider.authenticated != 0 {
		t.Fatalf("provider authenticated %d times", provider.authenticated)
	}
}

func TestServiceUnknownProvider(t *testing.T) {
	service, _ := newService(nil)
	if _, _, err := service.Begin("missing"); !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("Begin() error = %v", err)
	}
}

func TestOAuthTransactionStoreConsumesStateAtomically(t *testing.T) {
	store := newOAuthTransactionStore(time.Minute, 2)
	if err := store.Put(OAuthTransaction{ProviderID: "github", State: "state", CodeVerifier: "verifier"}); err != nil {
		t.Fatal(err)
	}

	var wait sync.WaitGroup
	results := make(chan bool, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, ok := store.Consume("state")
			results <- ok
		}()
	}
	wait.Wait()
	close(results)
	successes := 0
	for ok := range results {
		if ok {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful consumes = %d, want 1", successes)
	}
}

func TestOAuthTransactionStoreRejectsExpiredState(t *testing.T) {
	now := time.Now()
	store := newOAuthTransactionStore(time.Minute, 2)
	store.now = func() time.Time { return now }
	if err := store.Put(OAuthTransaction{ProviderID: "github", State: "state"}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	if _, ok := store.Consume("state"); ok {
		t.Fatal("Consume() accepted an expired transaction")
	}
}

func TestOAuthTransactionStoreRejectsNewTransactionAtCapacity(t *testing.T) {
	store := newOAuthTransactionStore(time.Minute, 2)
	for _, state := range []string{"first", "second"} {
		if err := store.Put(OAuthTransaction{ProviderID: "github", State: state}); err != nil {
			t.Fatalf("Put(%q) error = %v", state, err)
		}
	}

	if err := store.Put(OAuthTransaction{ProviderID: "github", State: "third"}); !errors.Is(err, ErrTransactionStoreFull) {
		t.Fatalf("Put() error = %v, want ErrTransactionStoreFull", err)
	}
	if _, ok := store.Consume("first"); !ok {
		t.Fatal("capacity rejection evicted an active transaction")
	}
	if _, ok := store.Consume("third"); ok {
		t.Fatal("capacity rejection retained the rejected transaction")
	}
}

func TestOAuthTransactionStoreReclaimsExpiredCapacity(t *testing.T) {
	now := time.Now()
	store := newOAuthTransactionStore(time.Minute, 1)
	store.now = func() time.Time { return now }
	if err := store.Put(OAuthTransaction{ProviderID: "github", State: "expired"}); err != nil {
		t.Fatal(err)
	}

	now = now.Add(time.Minute)
	if err := store.Put(OAuthTransaction{ProviderID: "github", State: "current"}); err != nil {
		t.Fatalf("Put() after expiry error = %v", err)
	}
	if _, ok := store.Consume("current"); !ok {
		t.Fatal("new transaction was not retained")
	}
}

func TestOAuthTransactionStoreConsumeRemovesOrderEntry(t *testing.T) {
	store := newOAuthTransactionStore(time.Minute, 1)
	if err := store.Put(OAuthTransaction{ProviderID: "github", State: "first"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Consume("first"); !ok {
		t.Fatal("Consume() did not find transaction")
	}
	if got := store.expiryQueue.Len(); got != 0 {
		t.Fatalf("expiry queue length = %d, want 0", got)
	}
	if err := store.Put(OAuthTransaction{ProviderID: "github", State: "second"}); err != nil {
		t.Fatalf("Put() after consume error = %v", err)
	}
}
