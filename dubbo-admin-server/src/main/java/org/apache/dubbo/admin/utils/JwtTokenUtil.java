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

package org.apache.dubbo.admin.utils;

import io.jsonwebtoken.Claims;
import io.jsonwebtoken.Jwts;
import io.jsonwebtoken.SignatureAlgorithm;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.util.Date;
import java.util.HashMap;
import java.util.Map;

/**
 * Jwt token tool class.
 */
@Component
public class JwtTokenUtil {
    /**
     * Jwt signingKey configurable
     */
    @Value("${admin.check.signSecret:}")
    public String secret;

    /**
     * token timeout configurable
     * default to be an hour: 1000 * 60 * 60
     */
    @Value("${admin.check.tokenTimeoutMilli:}")
    public long expiration;

    /**
     * default SignatureAlgorithm
     */
    public static final SignatureAlgorithm defaultAlgorithm = SignatureAlgorithm.HS512;

    /**
     * Generate the token
     *
     * @return token
     * @param rootUserName
     */
    public String generateToken(String rootUserName) {
        Map<String, Object> claims = new HashMap<>(1);
        claims.put("sub", rootUserName);
        return Jwts.builder()
                .setClaims(claims)
                .setExpiration(new Date(System.currentTimeMillis() + expiration))
                .setIssuedAt(new Date(System.currentTimeMillis()))
                .signWith(defaultAlgorithm, secret)
                .compact();
    }

    /**
     * Check whether the token is invalid
     *
     * @return boolean type
     * @param token
     */
    public Boolean canTokenBeExpiration(String token) {
        Claims claims;
        try {
            claims = Jwts.parser()
                    .setSigningKey(secret)
                    .parseClaimsJws(token)
                    .getBody();
            final Date exp = claims.getExpiration();
            if (exp.before(new Date(System.currentTimeMillis()))) {
                return false;
            }
            return true;
        } catch (Exception e) {
            return false;
        }
    }

}
