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
 * Obtain the API module list and the request parameters of the API parameter information interface.
 */
public class ApiInfoRequest {

    @ApiParam(value = "IP of Dubbo provider", required = true)
    private String dubboIp;

    @ApiParam(value = "Port of Dubbo provider", required = true)
    private String dubboPort;

    @ApiParam(value = "API full name (interface class full name. Method name), which must be passed when getting API parameter information")
    private String apiName;

    public String getDubboIp() {
        return dubboIp;
    }

    public void setDubboIp(String dubboIp) {
        this.dubboIp = dubboIp;
    }

    public String getDubboPort() {
        return dubboPort;
    }

    public void setDubboPort(String dubboPort) {
        this.dubboPort = dubboPort;
    }

    public String getApiName() {
        return apiName;
    }

    public void setApiName(String apiName) {
        this.apiName = apiName;
    }
}
