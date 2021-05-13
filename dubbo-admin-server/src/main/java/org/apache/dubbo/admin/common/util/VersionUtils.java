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

import org.apache.commons.lang3.StringUtils;
import org.apache.dubbo.admin.common.exception.ParamValidationException;
import org.apache.dubbo.admin.common.exception.VersionValidationException;
import org.apache.dubbo.admin.service.ConsumerService;
import org.apache.dubbo.admin.service.ProviderService;

public class VersionUtils {
    public static void versionCheck(String application,ProviderService providerService,
                                    ConsumerService consumerService,String msg) {
        if (StringUtils.isNotEmpty(application)) {
            String version = null;
            try {
                version = providerService.findVersionInApplication(application);
            } catch (Exception e) {
                if (e instanceof ParamValidationException) {
                    version = consumerService.findVersionInApplication(application);
                }
            }
            if ("2.6".equals(version)) {
                throw new VersionValidationException(msg);
            }
        }
    }
}
