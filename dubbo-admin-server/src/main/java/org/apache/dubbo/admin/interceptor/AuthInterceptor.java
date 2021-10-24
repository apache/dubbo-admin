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
package org.apache.dubbo.admin.interceptor;

import org.apache.dubbo.admin.authentication.InterceptorAuthentication;

import org.apache.dubbo.common.extension.ExtensionLoader;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Component;
import org.springframework.web.method.HandlerMethod;
import org.springframework.web.servlet.handler.HandlerInterceptorAdapter;

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import javax.validation.constraints.NotNull;
import java.util.Iterator;
import java.util.Set;

@Component
public class AuthInterceptor extends HandlerInterceptorAdapter {
    @Value("${admin.check.authority:true}")
    private boolean checkAuthority;

    @Override
    public boolean preHandle(HttpServletRequest request, HttpServletResponse response, Object handler) throws Exception {
        if (!(handler instanceof HandlerMethod) || !checkAuthority) {
            return true;
        }
        ExtensionLoader<InterceptorAuthentication> extensionLoader = ExtensionLoader.getExtensionLoader(InterceptorAuthentication.class);
        Set<InterceptorAuthentication> supportedExtensionInstances = extensionLoader.getSupportedExtensionInstances();
        Iterator<InterceptorAuthentication> iterator = supportedExtensionInstances.iterator();
        boolean flag = true;
        while (iterator.hasNext()) {
            InterceptorAuthentication interceptorAuthentication = iterator.next();
            boolean b = interceptorAuthentication.authentication(request, response, handler);
            flag = b & flag;
            if (!flag) {
                break;
            }
        }
        return flag;
    }

    public static void loginFailResponse(@NotNull HttpServletResponse response) {
        response.setStatus(HttpStatus.UNAUTHORIZED.value());
    }

    public static void authRejectedResponse(@NotNull HttpServletResponse response) {
        response.setStatus(HttpStatus.FORBIDDEN.value());
    }
}
