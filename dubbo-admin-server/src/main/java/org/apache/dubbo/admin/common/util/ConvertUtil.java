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

import org.apache.dubbo.admin.model.dto.BaseDTO;
import org.apache.dubbo.common.Constants;
import org.apache.dubbo.common.utils.StringUtils;

import java.util.HashMap;
import java.util.Map;

public class ConvertUtil {
    private ConvertUtil() {
    }

    public static Map<String, String> serviceName2Map(String serviceName) {
        String group = null;
        String version = null;
        int i = serviceName.indexOf("/");
        if (i > 0) {
            group = serviceName.substring(0, i);
            serviceName = serviceName.substring(i + 1);
        }
        i = serviceName.lastIndexOf(":");
        if (i > 0) {
            version = serviceName.substring(i + 1);
            serviceName = serviceName.substring(0, i);
        }

        Map<String, String> ret = new HashMap<String, String>();
        if (!StringUtils.isEmpty(serviceName)) {
            ret.put(Constants.INTERFACE_KEY, serviceName);
        }
        if (!StringUtils.isEmpty(version)) {
            ret.put(Constants.VERSION_KEY, version);
        }
        if (!StringUtils.isEmpty(group)) {
            ret.put(Constants.GROUP_KEY, group);
        }

        return ret;
    }

    public static String getIdFromDTO(BaseDTO baseDTO) {
        String id;
        if (StringUtils.isNotEmpty(baseDTO.getApplication())) {
            id = baseDTO.getApplication();
        } else {
            id = baseDTO.getService();
        }
        return id;
    }

    public static String getScopeFromDTO(BaseDTO baseDTO) {
        if (StringUtils.isNotEmpty(baseDTO.getApplication())) {
            return org.apache.dubbo.admin.common.util.Constants.APPLICATION;
        } else {
            return org.apache.dubbo.admin.common.util.Constants.SERVICE;
        }
    }

//    public static <T extends BaseDTO> T convertDTOtoStore(T dto) {
//        if (StringUtils.isNotEmpty(dto.getApplication())) {
//            dto.setScope("application");
//            dto.setKey(dto.getApplication());
//        } else {
//            dto.setScope("service");
//            dto.setKey(dto.getService());
//        }
//        return dto;
//    }
//
//    public static <T extends BaseDTO> T convertDTOtoDisplay(T dto) {
//        if (dto == null) {
//            return null;
//        }
//        if(dto.getScope().equals("application")) {
//            dto.setApplication(dto.getKey());
//        } else {
//            dto.setService(dto.getKey());
//        }
//        return dto;
//    }
}
