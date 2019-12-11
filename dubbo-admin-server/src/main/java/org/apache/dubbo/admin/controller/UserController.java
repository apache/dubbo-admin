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
package org.apache.dubbo.admin.controller;

import org.apache.dubbo.admin.annotation.Authority;

import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.context.request.RequestContextHolder;
import org.springframework.web.context.request.ServletRequestAttributes;

import javax.servlet.http.HttpServletRequest;
import java.util.Map;
import java.util.Objects;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;

@RestController
@RequestMapping("/api/{env}/user")
public class UserController {
    public static Map<String /*token*/, User /*user info*/> tokenMap = new ConcurrentHashMap<>();

    @Value("${admin.root.user.name:}")
    private String rootUserName;
    @Value("${admin.root.user.password:}")
    private String rootUserPassword;

    @RequestMapping(value = "/login", method = RequestMethod.GET)
    public String login(@RequestParam String userName, @RequestParam String password) {
        if (StringUtils.isBlank(rootUserName) || (rootUserName.equals(userName) && rootUserPassword.equals(password))) {
            UUID uuid = UUID.randomUUID();
            String token = uuid.toString();
            User user = new User();
            user.setUserName(userName);
            user.setLastUpdateTime(System.currentTimeMillis());
            tokenMap.put(token, user);
            return token;
        }
        return null;
    }

    @Authority(needLogin = true)
    @RequestMapping(value = "/logout", method = RequestMethod.DELETE)
    public boolean logout() {
        HttpServletRequest request =
                ((ServletRequestAttributes) Objects.requireNonNull(RequestContextHolder.getRequestAttributes())).getRequest();
        String token = request.getHeader("Authorization");
        return null != tokenMap.remove(token);
    }

    @Scheduled(cron= "0 5 * * * ?")
    public void clearExpiredToken() {
        tokenMap.entrySet().removeIf(entry -> entry.getValue() == null || System.currentTimeMillis() - entry.getValue().getLastUpdateTime() > 1000 * 60 * 15);
    }

    public static class User {
        private String userName;
        private long lastUpdateTime;

        public String getUserName() {
            return userName;
        }

        public void setUserName(String userName) {
            this.userName = userName;
        }

        public long getLastUpdateTime() {
            return lastUpdateTime;
        }

        public void setLastUpdateTime(long lastUpdateTime) {
            this.lastUpdateTime = lastUpdateTime;
        }
    }

}
