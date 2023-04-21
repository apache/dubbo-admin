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

package jwt

import (
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	IssuerKey     = "iss"
	SubjectKey    = "sub"
	CommonNameKey = "cn"
	ExpireKey     = "exp"
	ExtensionsKey = "ext"
)

type Claims struct {
	Subject    string
	Extensions string
	CommonName string
	ExpireTime int64
}

func NewClaims(subject, extensions, commonName string, cardinality int64) *Claims {
	return &Claims{
		Subject:    subject,
		Extensions: extensions,
		CommonName: commonName,
		ExpireTime: time.Now().Add(time.Duration(cardinality) * time.Millisecond).Unix(),
	}
}

func (t *Claims) Sign(pri *ecdsa.PrivateKey) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		IssuerKey:     "dubbo-authority",
		SubjectKey:    t.Subject,
		CommonNameKey: t.CommonName,
		ExpireKey:     t.ExpireTime,
		ExtensionsKey: t.Extensions,
	}).SignedString(pri)
}

func Verify(pub *ecdsa.PublicKey, token string) (*Claims, error) {
	claims, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return pub, nil
	})
	if err != nil {
		return nil, err
	}
	return &Claims{
		Subject:    claims.Claims.(jwt.MapClaims)[SubjectKey].(string),
		Extensions: claims.Claims.(jwt.MapClaims)[ExtensionsKey].(string),
		CommonName: claims.Claims.(jwt.MapClaims)[CommonNameKey].(string),
		ExpireTime: int64(claims.Claims.(jwt.MapClaims)[ExpireKey].(float64)),
	}, nil
}
