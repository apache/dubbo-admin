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
	"container/list"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
)

var (
	ErrProviderNotFound     = errors.New("authentication provider not found")
	ErrProviderMismatch     = errors.New("OAuth callback provider does not match transaction")
	ErrStateMismatch        = errors.New("OAuth callback state does not match transaction")
	ErrTransactionGone      = errors.New("OAuth transaction is missing, expired, or already consumed")
	ErrTransactionStoreFull = errors.New("OAuth transaction store is full")
)

const (
	oauthTransactionTTL      = 10 * time.Minute
	oauthTransactionCapacity = 10_000
)

// Service coordinates provider registration and the shared OAuth callback flow.
type Service struct {
	providers    map[string]Provider
	public       []PublicProvider
	transactions *oauthTransactionStore
}

func NewService(ctx context.Context, configs map[string]configauth.ProviderConfig, client *http.Client) (*Service, error) {
	ids := make([]string, 0, len(configs))
	for id := range configs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	providers := make([]Provider, 0, len(ids))
	for _, id := range ids {
		cfg := configs[id]
		switch cfg.Type {
		case configauth.ProviderTypeGitHub:
			providers = append(providers, NewGitHubProvider(id, cfg))
		case configauth.ProviderTypeOIDC:
			provider, err := NewOIDCProvider(ctx, id, cfg, client)
			if err != nil {
				return nil, err
			}
			providers = append(providers, provider)
		default:
			return nil, fmt.Errorf("unsupported authentication provider type %q", cfg.Type)
		}
	}
	return newService(providers)
}

func newService(providers []Provider) (*Service, error) {
	service := &Service{
		providers:    make(map[string]Provider, len(providers)),
		transactions: newOAuthTransactionStore(oauthTransactionTTL, oauthTransactionCapacity),
	}
	for _, provider := range providers {
		if provider == nil || provider.ID() == "" {
			return nil, errors.New("authentication provider must have an ID")
		}
		if _, exists := service.providers[provider.ID()]; exists {
			return nil, fmt.Errorf("duplicate authentication provider %q", provider.ID())
		}
		service.providers[provider.ID()] = provider
		service.public = append(service.public, PublicProvider{
			ID: provider.ID(), DisplayName: provider.DisplayName(), IconURL: provider.IconURL(),
		})
	}
	sort.Slice(service.public, func(i, j int) bool { return service.public[i].ID < service.public[j].ID })
	return service, nil
}

func NewServiceFromProviders(providers ...Provider) (*Service, error) {
	return newService(providers)
}

func (s *Service) PublicProviders() []PublicProvider {
	return append([]PublicProvider{}, s.public...)
}

func (s *Service) Begin(providerID string) (OAuthTransaction, string, error) {
	provider, ok := s.providers[providerID]
	if !ok {
		return OAuthTransaction{}, "", ErrProviderNotFound
	}
	state, err := randomBase64URL(32)
	if err != nil {
		return OAuthTransaction{}, "", fmt.Errorf("generate OAuth state: %w", err)
	}
	verifier, err := randomBase64URL(32)
	if err != nil {
		return OAuthTransaction{}, "", fmt.Errorf("generate PKCE verifier: %w", err)
	}
	transaction := OAuthTransaction{ProviderID: providerID, State: state, CodeVerifier: verifier}
	if provider.NeedsNonce() {
		transaction.Nonce, err = randomBase64URL(32)
		if err != nil {
			return OAuthTransaction{}, "", fmt.Errorf("generate OIDC nonce: %w", err)
		}
	}
	if err := s.transactions.Put(transaction); err != nil {
		return OAuthTransaction{}, "", err
	}
	return transaction, provider.AuthorizationURL(transaction), nil
}

func (s *Service) Complete(ctx context.Context, providerID, state, code string, transaction OAuthTransaction) (Principal, error) {
	if providerID != transaction.ProviderID {
		return Principal{}, ErrProviderMismatch
	}
	if subtle.ConstantTimeCompare([]byte(state), []byte(transaction.State)) != 1 {
		return Principal{}, ErrStateMismatch
	}
	storedTransaction, ok := s.transactions.Consume(state)
	if !ok {
		return Principal{}, ErrTransactionGone
	}
	if providerID != storedTransaction.ProviderID {
		return Principal{}, ErrProviderMismatch
	}
	provider, ok := s.providers[providerID]
	if !ok {
		return Principal{}, ErrProviderNotFound
	}
	if code == "" {
		return Principal{}, errors.New("OAuth callback code is missing")
	}
	return provider.Authenticate(ctx, code, storedTransaction.CodeVerifier, storedTransaction.Nonce)
}

func (s *Service) PostLoginRedirectURL(providerID string) (string, error) {
	provider, ok := s.providers[providerID]
	if !ok {
		return "", ErrProviderNotFound
	}
	return provider.PostLoginRedirectURL(), nil
}

func PKCEChallenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func randomBase64URL(size int) (string, error) {
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// TODO: can move it to a seperated file?
type storedOAuthTransaction struct {
	transaction OAuthTransaction
	expiresAt   time.Time
}

// TODO: move this store into redis?
type oauthTransactionStore struct {
	mu          sync.Mutex
	ttl         time.Duration
	capacity    int
	now         func() time.Time
	items       map[string]*list.Element
	expiryQueue *list.List
}

func newOAuthTransactionStore(ttl time.Duration, capacity int) *oauthTransactionStore {
	return &oauthTransactionStore{
		ttl: ttl, capacity: capacity, now: time.Now,
		items: make(map[string]*list.Element), expiryQueue: list.New(),
	}
}

func (s *oauthTransactionStore) Put(transaction OAuthTransaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	s.removeExpired(now)
	if existing, ok := s.items[transaction.State]; ok {
		s.remove(existing)
	}
	if len(s.items) >= s.capacity {
		return ErrTransactionStoreFull
	}
	element := s.expiryQueue.PushBack(storedOAuthTransaction{transaction: transaction, expiresAt: now.Add(s.ttl)})
	s.items[transaction.State] = element
	return nil
}

func (s *oauthTransactionStore) Consume(state string) (OAuthTransaction, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	element, ok := s.items[state]
	if !ok {
		return OAuthTransaction{}, false
	}
	stored := element.Value.(storedOAuthTransaction)
	s.remove(element)
	if !stored.expiresAt.After(s.now()) {
		return OAuthTransaction{}, false
	}
	return stored.transaction, true
}

func (s *oauthTransactionStore) removeExpired(now time.Time) {
	for element := s.expiryQueue.Front(); element != nil; element = s.expiryQueue.Front() {
		stored := element.Value.(storedOAuthTransaction)
		if stored.expiresAt.After(now) {
			return
		}
		s.remove(element)
	}
}

func (s *oauthTransactionStore) remove(element *list.Element) {
	stored := element.Value.(storedOAuthTransaction)
	delete(s.items, stored.transaction.State)
	s.expiryQueue.Remove(element)
}
