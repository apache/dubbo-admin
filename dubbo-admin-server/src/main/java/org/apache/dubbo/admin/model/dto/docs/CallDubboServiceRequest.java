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
package org.apache.dubbo.admin.model.dto.docs;

import io.swagger.annotations.ApiParam;


/**
 * Call Dubbo api to request parameters.
 */
public class CallDubboServiceRequest {

    @ApiParam(value = "Address of registration center, such as: nacos://127.0.0.1:8848", required = true)
    private String registryCenterUrl;

    @ApiParam(value = "Dubbo interface full package path", required = true)
    private String interfaceClassName;

    @ApiParam(value = "Method name of the service", required = true)
    private String methodName;

    @ApiParam(value = "Whether to call asynchronously, false by default")
    private boolean async = false;

    @ApiParam(value = "The version of API")
    private String version;

    public String getVersion() {
        return version;
    }

    public void setVersion(String version) {
        this.version = version;
    }

    public String getRegistryCenterUrl() {
        return registryCenterUrl;
    }

    public void setRegistryCenterUrl(String registryCenterUrl) {
        this.registryCenterUrl = registryCenterUrl;
    }

    public String getInterfaceClassName() {
        return interfaceClassName;
    }

    public void setInterfaceClassName(String interfaceClassName) {
        this.interfaceClassName = interfaceClassName;
    }

    public String getMethodName() {
        return methodName;
    }

    public void setMethodName(String methodName) {
        this.methodName = methodName;
    }

    public boolean isAsync() {
        return async;
    }

    public void setAsync(boolean async) {
        this.async = async;
    }
}
