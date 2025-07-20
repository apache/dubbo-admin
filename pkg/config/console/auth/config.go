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
	"errors"

	"github.com/apache/dubbo-admin/pkg/config"
)

const DefaultExpirationTime = 7200

// Config AuthConfig configure the valid user and password
type Config struct {
	config.BaseConfig
	User           string `json:"user"`
	Password       string `json:"password"`
	ExpirationTime int    `json:"expirationTime"`
}

func (c *Config) Validate() error {
	if c.User == "" || c.Password == "" {
		return errors.New("auth: user or password is needed, but found empty")
	}
	if c.ExpirationTime <= 0 || c.ExpirationTime >= 24*60*60 {
		return errors.New("auth: expirationTime should be greater than 0 and less than 86400")
	}
	return nil
}
