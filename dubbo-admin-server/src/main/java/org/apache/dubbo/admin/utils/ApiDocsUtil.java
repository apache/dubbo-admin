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

import org.apache.commons.lang3.StringUtils;

/**
 * uitls for dubbo api docs.
 *
 * @author klw(213539 @ qq.com)
 * @date 2020/11/10 10:48
 */
public class ApiDocsUtil {

    /**
     * Simply check whether a string is in JSON format.
     * 2020/11/13 9:44
     * @param str a string to check
     * @return boolean
     */
    public static boolean isJsonStr(String str) {
        boolean flag = false;
        if (StringUtils.isNotBlank(str)) {
            str = str.trim();
            if (str.startsWith("{") && str.endsWith("}")) {
                flag = true;
            } else if (str.startsWith("[") && str.endsWith("]")) {
                flag = true;
            }
        }
        return flag;
    }

}
