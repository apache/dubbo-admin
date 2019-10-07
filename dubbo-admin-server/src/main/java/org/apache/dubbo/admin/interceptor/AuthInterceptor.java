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

import org.apache.dubbo.admin.annotation.Authority;
import org.apache.dubbo.admin.controller.UserController;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Component;
import org.springframework.web.method.HandlerMethod;
import org.springframework.web.servlet.handler.HandlerInterceptorAdapter;

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.lang.reflect.Method;

@Component
public class AuthInterceptor extends HandlerInterceptorAdapter {
    @Value("${admin.check.authority:true}")
    private boolean checkAuthority;
    @Override
    public boolean preHandle(HttpServletRequest request, HttpServletResponse response, Object handler) throws Exception {
        if (!(handler instanceof HandlerMethod) || !checkAuthority) {
            return true;
        }
        HandlerMethod handlerMethod = (HandlerMethod) handler;
        Method method = handlerMethod.getMethod();
        Authority authority = method.getDeclaredAnnotation(Authority.class);
        if (null == authority) {
            authority = method.getDeclaringClass().getDeclaredAnnotation(Authority.class);
        }
        if (null != authority && authority.needLogin()) {
            String authorization = request.getHeader("Authorization");
            UserController.User user = UserController.tokenMap.get(authorization);
            if (null != user && System.currentTimeMillis() - user.getLastUpdateTime() <= 1000 * 60 * 15) {
                user.setLastUpdateTime(System.currentTimeMillis());
                return true;
            }
            response.setStatus(HttpStatus.UNAUTHORIZED.value());
            return false;
        } else {
            return true;
        }
    }
}
