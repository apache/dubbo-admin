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
	"errors"
	"net/http"
	"slices"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
	consoleauth "github.com/apache/dubbo-admin/pkg/console/auth"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
)

type AuthHandler struct {
	config  *configauth.Config
	service *consoleauth.Service
}

type providersResponse struct {
	Methods   []string                     `json:"methods"`
	Providers []consoleauth.PublicProvider `json:"providers"`
}

func NewAuthHandler(ctx consolectx.Context) (*AuthHandler, error) {
	config := ctx.Config().Console.Auth
	service, err := consoleauth.NewService(ctx.AppContext(), config.Providers, nil)
	if err != nil {
		return nil, err
	}
	return newAuthHandler(config, service), nil
}

func newAuthHandler(config *configauth.Config, service *consoleauth.Service) *AuthHandler {
	return &AuthHandler{config: config, service: service}
}

func (h *AuthHandler) Login(c *gin.Context) {
	if !slices.Contains(h.config.Methods, configauth.MethodPassword) {
		c.JSON(http.StatusNotFound, model.NewBizErrorResp(bizerror.New(bizerror.NotFoundError, "password login is not enabled")))
		return
	}
	user := c.PostForm("user")
	password := c.PostForm("password")
	if user != h.config.User || password != h.config.Password {
		c.JSON(http.StatusUnauthorized, model.NewBizErrorResp(bizerror.New(bizerror.Unauthorized, "username or password is not correct!")))
		return
	}
	session := sessions.Default(c)
	if err := consoleauth.PutPrincipal(session, consoleauth.LocalPrincipal(user)); err != nil {
		writeSessionError(c, err)
		return
	}
	if err := session.Save(); err != nil {
		writeSessionError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.NewSuccessResp(true))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Options(sessions.Options{
		Path: "/", MaxAge: -1, Secure: h.config.SessionCookieSecure, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	if err := session.Save(); err != nil {
		writeSessionError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.NewSuccessResp(true))
}

func (h *AuthHandler) Providers(c *gin.Context) {
	c.JSON(http.StatusOK, model.NewSuccessResp(providersResponse{
		Methods: append([]string{}, h.config.Methods...), Providers: h.service.PublicProviders(),
	}))
}

func (h *AuthHandler) ProviderLogin(c *gin.Context) {
	transaction, authorizationURL, err := h.service.Begin(c.Param("provider"))
	if err != nil {
		writeProviderError(c, err)
		return
	}
	session := sessions.Default(c)
	if err := consoleauth.PutOAuthTransaction(session, transaction); err != nil {
		writeSessionError(c, err)
		return
	}
	if err := session.Save(); err != nil {
		writeSessionError(c, err)
		return
	}
	c.Redirect(http.StatusFound, authorizationURL)
}

func (h *AuthHandler) ProviderCallback(c *gin.Context) {
	session := sessions.Default(c)
	transaction, err := consoleauth.ConsumeOAuthTransaction(session)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument, err.Error())))
		return
	}
	// Clear the client-side transaction reference before contacting the Provider.
	if err := session.Save(); err != nil {
		writeSessionError(c, err)
		return
	}
	principal, err := h.service.Complete(c.Request.Context(), c.Param("provider"), c.Query("state"), c.Query("code"), transaction)
	if err != nil {
		writeProviderError(c, err)
		return
	}
	if err := consoleauth.PutPrincipal(session, principal); err != nil {
		writeSessionError(c, err)
		return
	}
	if err := session.Save(); err != nil {
		writeSessionError(c, err)
		return
	}
	redirectURL, err := h.service.PostLoginRedirectURL(transaction.ProviderID)
	if err != nil {
		writeProviderError(c, err)
		return
	}
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *AuthHandler) UserInfo(c *gin.Context) {
	principal, ok := consoleauth.PrincipalFromContext(c)
	if !ok {
		writeUnauthorized(c)
		return
	}
	c.JSON(http.StatusOK, model.NewSuccessResp(principal))
}

func writeProviderError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	code := bizerror.InvalidArgument
	if errors.Is(err, consoleauth.ErrProviderNotFound) {
		status = http.StatusNotFound
		code = bizerror.NotFoundError
	} else if errors.Is(err, consoleauth.ErrTransactionStoreFull) {
		status = http.StatusTooManyRequests
	}
	c.JSON(status, model.NewBizErrorResp(bizerror.New(code, err.Error())))
}

func writeSessionError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, model.NewBizErrorResp(bizerror.New(bizerror.SessionError, err.Error())))
}

func writeUnauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, model.NewBizErrorResp(bizerror.NewUnauthorizedError()))
}
