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

package org.apache.dubbo.admin.data.config.impl;

import com.ctrip.framework.apollo.openapi.client.ApolloOpenApiClient;
import org.apache.dubbo.admin.data.config.GovernanceConfiguration;
import org.apache.dubbo.common.URL;
import org.springframework.beans.factory.annotation.Value;

public class ApolloConfiguration implements GovernanceConfiguration {

    @Value("${dubbo.apollo.token}")
    private String token;
    private URL url;
    private ApolloOpenApiClient client;


    @Override
    public void setUrl(URL url) {
       this.url = url;
    }

    @Override
    public URL getUrl() {
        return url;
    }

    @Override
    public void init() {
        client = ApolloOpenApiClient.newBuilder().withPortalUrl(url.getAddress()).withToken(token).build();
    }

    @Override
    public String setConfig(String key, String value) {
        return null;
    }

    @Override
    public String getConfig(String key) {
        return null;
    }

    @Override
    public boolean deleteConfig(String key) {
        return false;
    }
}
