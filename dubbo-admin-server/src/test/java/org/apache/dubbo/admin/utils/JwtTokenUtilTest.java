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
import org.junit.Test;

import java.util.Date;


public class JwtTokenUtilTest {
    private final String secret = "a1g2y47dg3dj59fjhhsd7cnewy73j";
    public String token;

    @Test
    public void generateTokenTest() {
        token = JwtTokenUtil.generateToken("test");
        System.out.println(token);
    }

    @Test
    public void canTokenBeExpirationTest() {
        generateTokenTest();
        Boolean aBoolean = JwtTokenUtil.canTokenBeExpiration(token);
        Claims claims = Jwts.parser()
                .setSigningKey(secret)
                .parseClaimsJws(token)
                .getBody();
        Date iat = claims.getIssuedAt();
        Date exp = claims.getExpiration();

        System.out.println(aBoolean);
        System.out.println(iat);
        System.out.println(exp);
        System.out.println(exp.getTime()-iat.getTime());
    }

    @Test
    public void refreshTokenTest() {
        generateTokenTest();
        String refreshToken = JwtTokenUtil.refreshToken(token);
        Claims claims_token = Jwts.parser()
                .setSigningKey(secret)
                .parseClaimsJws(token)
                .getBody();
        Claims claims_refreshToken = Jwts.parser()
                .setSigningKey(secret)
                .parseClaimsJws(refreshToken)
                .getBody();
        System.out.println(claims_refreshToken.getIssuedAt());
        System.out.println(claims_refreshToken.getExpiration());
//        System.out.println(claims_token.getExpiration());
    }

}
