// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jwt_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/apache/dubbo-admin/pkg/core/jwt"

	v4 "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	t.Parallel()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.Nil(t, err)

	token, err := jwt.NewClaims("test", "test", "test123", 60*1000).Sign(key)
	assert.Nil(t, err)

	claims, err := jwt.Verify(&key.PublicKey, token)

	assert.Nil(t, err)

	assert.NotNil(t, claims)
	assert.Equal(t, "test", claims.Subject)
	assert.Equal(t, "test123", claims.CommonName)
	assert.Equal(t, "test", claims.Extensions)
}

func TestVerifyFailed(t *testing.T) {
	t.Parallel()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Nil(t, err)

	token, err := v4.NewWithClaims(v4.SigningMethodRS256, v4.MapClaims{
		"iss": "dubbo-authority",
		"sub": "test",
		"exp": time.Now().Add(time.Duration(10*3600) * time.Millisecond).UnixMilli(),
		"ext": "test",
	}).SignedString(key)
	assert.Nil(t, err)

	claims, err := jwt.Verify(nil, token)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "Unexpected signing method")
}
