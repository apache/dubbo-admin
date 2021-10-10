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

import java.util.Date;
import java.util.HashMap;
import java.util.Map;

/**
 * Jwt token tool class.
 */
public class JwtTokenUtil {
    /**
     * Jwt signingKey configurable
     */
    public static String defaultSecret = "86295dd0c4ef69a1036b0b0c15158d77";

    /**
     * token timeout configurable
     * default to be an hour: 1000 * 60 * 60
     */
    public static long defaultExpiration = 1000 * 60 * 60;

    /**
     * default SignatureAlgorithm
     */
    public static final SignatureAlgorithm defaultAlgorithm = SignatureAlgorithm.HS512;

    /**
     * Generate the token
     *
     * @return token
     * @param rootUserName
     * @param secret
     * @param expiration
     */
    public static String generateToken(String rootUserName, String secret, long expiration) {
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
     * Generate the token
     *
     * @return token
     * @param rootUserName
     * @param secret
     */
    public static String generateToken(String rootUserName, String secret) {
        defaultSecret = secret;
        return generateToken(rootUserName, secret, defaultExpiration);
    }

    /**
     * Generate the token
     *
     * @return token
     * @param rootUserName
     * @param expiration
     */
    public static String generateToken(String rootUserName, long expiration) {
        defaultExpiration = expiration;
        return generateToken(rootUserName, defaultSecret, expiration);
    }

    /**
     * Generate the token
     *
     * @return token
     * @param rootUserName
     */
    public static String generateToken(String rootUserName) {
        return generateToken(rootUserName, defaultSecret, defaultExpiration);
    }

    /**
     * Check whether the token is invalid
     *
     * @return boolean type
     * @param token
     */
    public static Boolean canTokenBeExpiration(String token) {
        Claims claims;
        try {
            claims = Jwts.parser()
                    .setSigningKey(defaultSecret)
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
