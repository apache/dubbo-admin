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

import static org.hamcrest.CoreMatchers.is;
import static org.junit.Assert.assertThat;


public class JwtTokenUtilTest {
    public final String defaultSecret = "86295dd0c4ef69a1036b0b0c15158d77";
    public final long defaultExpire = 1000 * 60 * 60;
    public String testToken = "eyJhbGciOiJIUzUxMiJ9.eyJleHAiOjE2MzM4NTI2" +
            "MDQsInN1YiI6InRlc3QiLCJpYXQiOjE2MzM4NDkwMDR9.e1UqT-3W3EZcI6" +
            "Dt-35b0Q_MA9ZhARAq59ZvkOYNlWL0Fa-RFk1ZQKs15Hk7LATfVH2DAo0JL" +
            "rHcY-79jDFnfQ";
    public long testIat = 1633849279000L;
    public long testExp = 1633852879000L;
    public String userName = "test";

}
