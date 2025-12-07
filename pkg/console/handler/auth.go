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

package handler

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
)

func Login(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.PostForm("user")
		password := c.PostForm("password")
		// verify username and password
		authCfg := ctx.Config().Console.Auth
		if user != authCfg.User || password != authCfg.Password {
			authErr := bizerror.New(bizerror.Unauthorized, "username or password is not correct!")
			c.JSON(http.StatusUnauthorized, model.NewBizErrorResp(authErr))
			return
		}
		session := sessions.Default(c)
		session.Set("user", user)
		session.Options(sessions.Options{
			MaxAge: authCfg.ExpirationTime,
			Path:   "/",
		})
		err := session.Save()
		if err != nil {
			sessionErr := bizerror.New(bizerror.SessionError, err.Error())
			c.JSON(http.StatusOK, model.NewBizErrorResp(sessionErr))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(true))
	}
}

func Logout(_ consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		session.Clear()
		err := session.Save()
		if err != nil {
			sessionErr := bizerror.New(bizerror.SessionError, err.Error())
			c.JSON(http.StatusOK, model.NewBizErrorResp(sessionErr))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(true))
	}
}
