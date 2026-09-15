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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-contrib/sessions"
)

const (
	PrincipalSessionKey        = "principal"
	LegacyUserSessionKey       = "user"
	OAuthTransactionSessionKey = "oauth_transaction"
)

type OAuthTransaction struct {
	ProviderID   string `json:"providerId"`
	State        string `json:"state"`
	CodeVerifier string `json:"codeVerifier"`
	Nonce        string `json:"nonce,omitempty"`
}

func PutPrincipal(session sessions.Session, principal Principal) error {
	raw, err := json.Marshal(principal)
	if err != nil {
		return fmt.Errorf("marshal principal: %w", err)
	}
	session.Set(PrincipalSessionKey, string(raw))
	session.Delete(LegacyUserSessionKey)
	return nil
}

func PrincipalFromSession(session sessions.Session) (Principal, error) {
	if value := session.Get(PrincipalSessionKey); value != nil {
		raw, ok := sessionJSONString(value)
		if !ok {
			return Principal{}, fmt.Errorf("invalid principal session value %T", value)
		}
		var principal Principal
		if err := json.Unmarshal([]byte(raw), &principal); err != nil {
			return Principal{}, fmt.Errorf("decode principal session: %w", err)
		}
		if principal.Subject == "" {
			return Principal{}, ErrNoPrincipal
		}
		if principal.Groups == nil {
			principal.Groups = []string{}
		}
		if principal.Roles == nil {
			principal.Roles = []string{}
		}
		return principal, nil
	}
	legacyUser, ok := session.Get(LegacyUserSessionKey).(string)
	if !ok || legacyUser == "" {
		return Principal{}, ErrNoPrincipal
	}
	return LocalPrincipal(legacyUser), nil
}

func PutOAuthTransaction(session sessions.Session, transaction OAuthTransaction) error {
	reference := OAuthTransaction{ProviderID: transaction.ProviderID, State: transaction.State}
	raw, err := json.Marshal(reference)
	if err != nil {
		return fmt.Errorf("marshal OAuth transaction: %w", err)
	}
	session.Set(OAuthTransactionSessionKey, string(raw))
	return nil
}

func ConsumeOAuthTransaction(session sessions.Session) (OAuthTransaction, error) {
	value := session.Get(OAuthTransactionSessionKey)
	session.Delete(OAuthTransactionSessionKey)
	if value == nil {
		return OAuthTransaction{}, errors.New("OAuth transaction is missing or already consumed")
	}
	raw, ok := sessionJSONString(value)
	if !ok {
		return OAuthTransaction{}, fmt.Errorf("invalid OAuth transaction session value %T", value)
	}
	var transaction OAuthTransaction
	if err := json.Unmarshal([]byte(raw), &transaction); err != nil {
		return OAuthTransaction{}, fmt.Errorf("decode OAuth transaction: %w", err)
	}
	return transaction, nil
}

func sessionJSONString(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case []byte:
		return string(typed), true
	default:
		return "", false
	}
}
