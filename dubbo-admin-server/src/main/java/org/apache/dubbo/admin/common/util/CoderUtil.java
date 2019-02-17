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

package org.apache.dubbo.admin.common.util;

import org.apache.dubbo.common.io.Bytes;
import org.apache.dubbo.common.logger.Logger;
import org.apache.dubbo.common.logger.LoggerFactory;

import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;

public class CoderUtil {

    private static final Logger logger = LoggerFactory.getLogger(CoderUtil.class);
    private static MessageDigest md;
    private static final char[] hexCode = "0123456789ABCDEF".toCharArray();

    static {
        try {
            md = MessageDigest.getInstance("MD5");
        } catch (NoSuchAlgorithmException e) {
            logger.error(e.getMessage(), e);
        }
    }

    public static String MD5_16bit(String input) {
        String hash = MD5_32bit(input);
        if (hash == null) {
            return null;
        }
        return hash.substring(8, 24);
    }

    public static String MD5_32bit(String input) {
        if (input == null || input.length() == 0) {
            return null;
        }
        md.update(input.getBytes());
        byte[] digest = md.digest();
        String hash = convertToString(digest);
        return hash;
    }

    public static String MD5_32bit(byte[] input) {
        if (input == null || input.length == 0) {
            return null;
        }
        md.update(input);
        byte[] digest = md.digest();
        String hash = convertToString(digest);
        return hash;
    }

    private static String convertToString(byte[] data) {
        StringBuilder r = new StringBuilder(data.length * 2);
        for (byte b : data) {
            r.append(hexCode[(b >> 4) & 0xF]);
            r.append(hexCode[(b & 0xF)]);
        }
        return r.toString();
    }

    public static String decodeBase64(String source) {
        return new String(Bytes.base642bytes(source));
    }
}
