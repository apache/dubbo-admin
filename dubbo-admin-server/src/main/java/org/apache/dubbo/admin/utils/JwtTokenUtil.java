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
     * Jwt signingKey
     */
    private static final String secret = "a1g2y47dg3dj59fjhhsd7cnewy73j";

    /**
     * token timeout configurable
     * default to be an hour: 1000 * 60 * 60
     */
    private static final long expiration = 60 * 60 * 1000;

    /**
     * initialise jwt token and claims
     *
     * @return jwt token
     * @param rootUserName
     */
    public static String generateToken(String rootUserName) {
        Map<String, Object> claims = new HashMap<>(1);
        claims.put("sub", rootUserName);
        return generateToken(claims);
    }

    /**
     * generate token
     *
     * @return jwt token
     * @param claims
     */
    private static String generateToken(Map<String, Object> claims) {
        return Jwts.builder()
                .setClaims(claims)
                .setExpiration(generateExpirationDate())
                .setIssuedAt(generateCurrentDate())
                .signWith(SignatureAlgorithm.HS512, secret)
                .compact();
    }

    /**
     * Get the current time
     */
    private static Date generateCurrentDate() {
        return new Date(System.currentTimeMillis());
    }

    /**
     * Get the token expiration time
     */
    private static Date generateExpirationDate() {
        return new Date(System.currentTimeMillis() + expiration);
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
                    .setSigningKey(secret)
                    .parseClaimsJws(token)
                    .getBody();
            final Date exp = claims.getExpiration();
            if (exp.before(generateCurrentDate())) {
                return false;
            }
            return true;
        } catch (Exception e) {
            return false;
        }
    }
    
    /**
     * refresh token
     *
     * @return token
     * @param token
     */
    public static String refreshToken(String token) {
        String refreshedToken;
        try {
            final Claims claims = Jwts.parser()
                    .setSigningKey(secret)
                    .parseClaimsJws(token)
                    .getBody();
            refreshedToken = generateToken(claims);
        } catch (Exception e) {
            refreshedToken = null;
        }
        return refreshedToken;
    }

}
