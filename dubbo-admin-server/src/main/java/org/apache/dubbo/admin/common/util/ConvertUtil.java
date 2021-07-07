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
import org.apache.dubbo.common.utils.StringUtils;
import org.apache.dubbo.metadata.definition.model.MethodDefinition;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static org.apache.dubbo.admin.common.util.Constants.COLON;

public class ConvertUtil {
    private ConvertUtil() {
    }

    public static Map<String, String> serviceName2Map(String serviceName) {
        String group = Tool.getGroup(serviceName);
        String version = Tool.getVersion(serviceName);
        String interfaze = Tool.getInterface(serviceName);

        Map<String, String> ret = new HashMap<String, String>();
        if (!StringUtils.isEmpty(serviceName)) {
            ret.put(Constants.INTERFACE_KEY, interfaze);
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
        if (StringUtils.isNotEmpty(baseDTO.getApplication())) {
            return baseDTO.getApplication();
        }
        // id format: "${class}:${version}:${group}"
        return new StringBuilder(baseDTO.getService()).append(COLON).append(null2EmptyString(baseDTO.getServiceVersion()))
                .append(COLON).append(null2EmptyString(baseDTO.getServiceGroup())).toString();
    }

    /**
     * Detach interface class, version and group from id.
     * @param id
     * @return java.lang.String[] 0: interface class; 1: version; 2: group
     */
    public static String[] detachId(String id) {
        if (id.contains(COLON)) {
            return id.split(COLON);
        } else {
            return new String[]{id};
        }
    }

    public static void detachIdToService(String id, BaseDTO baseDTO) {
        String[] detachResult = detachId(id);
        baseDTO.setService(detachResult[0]);
        if (detachResult.length > 1) {
            baseDTO.setServiceVersion(detachResult[1]);
        }
        if (detachResult.length > 2) {
            baseDTO.setServiceGroup(detachResult[2]);
        }
    }

    public static String getServiceIdFromDTO(BaseDTO baseDTO, String serviceVersion, String serviceGroup,
                                             boolean groupAsFolder) {
        StringBuilder buf = new StringBuilder();
        buf.append(baseDTO.getService());
        if (StringUtils.isNotEmpty(serviceVersion)) {
            buf.append(COLON).append(serviceVersion);
        }
        if (StringUtils.isNotEmpty(serviceGroup)) {
            if (groupAsFolder) {
                buf.insert(0, serviceGroup + "/");
            } else {
                buf.append(COLON).append(serviceGroup);
            }
        }
        return buf.toString();
    }

    public static String null2EmptyString(String str) {
        if (null == str) {
            str = StringUtils.EMPTY_STRING;
        }
        return str;
    }

    public static String getScopeFromDTO(BaseDTO baseDTO) {
        if (StringUtils.isNotEmpty(baseDTO.getApplication())) {
            return org.apache.dubbo.admin.common.util.Constants.APPLICATION;
        } else {
            return org.apache.dubbo.admin.common.util.Constants.SERVICE;
        }
    }

    public static Map methodList2Map(List<MethodDefinition> methods) {
        Map<String, MethodDefinition> res = new HashMap<>();
        for (int i = 0; i < methods.size(); i++) {
            res.put(methods.get(i).getName(), methods.get(i));
        }
        return res;
    }
}
