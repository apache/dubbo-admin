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

package org.apache.dubbo.admin.registry.config.impl;

import com.ctrip.framework.apollo.openapi.client.ApolloOpenApiClient;
import com.ctrip.framework.apollo.openapi.dto.OpenItemDTO;

import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.registry.config.GovernanceConfiguration;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.extension.SPI;
import org.apache.dubbo.common.utils.StringUtils;
import org.springframework.beans.factory.annotation.Value;

import java.util.Arrays;
import java.util.stream.Collectors;

import static org.apache.dubbo.common.constants.CommonConstants.COMMA_SPLIT_PATTERN;

@SPI("apollo")
public class ApolloConfiguration implements GovernanceConfiguration {

    private static final String APOLLO_ENV_KEY = "env";
    private static final String CLUSTER_KEY = "cluster";
    private static final String TOKEN_KEY = "token";
    private static final String APOLLO_APPID_KEY = "app.id";
    private static final String APOLLO_PROTOCOL_PREFIX = "http://";

    @Value("${admin.apollo.token:}")
    private String configToken;

    @Value("${admin.apollo.cluster:}")
    private String configCluster;

    @Value("${admin.apollo.namespace:}")
    private String configNamespace;

    @Value("${admin.apollo.env:}")
    private String configEnv;

    @Value("${admin.apollo.appId:}")
    private String configAppId;

    private String token;
    private String cluster;
    private String namespace;
    private String env;
    private String appId;
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
        token = url.getParameter(TOKEN_KEY, configToken);
        cluster = url.getParameter(CLUSTER_KEY, configCluster);
        namespace = url.getParameter(Constants.NAMESPACE_KEY, configNamespace);
        env = url.getParameter(APOLLO_ENV_KEY, configEnv);
        appId = url.getParameter(APOLLO_APPID_KEY, configAppId);
        String address = getAddressWithProtocolPrefix(url);
        client = ApolloOpenApiClient.newBuilder().withPortalUrl(address).withToken(token).build();
    }
    private String getAddressWithProtocolPrefix(URL url) {
        String address = url.getBackupAddress();
        if (StringUtils.isNotEmpty(address)) {
            address = Arrays.stream(COMMA_SPLIT_PATTERN.split(address))
                    .map(addr -> {
                        if (addr.startsWith(APOLLO_PROTOCOL_PREFIX)) {
                            return addr;
                        }
                        return APOLLO_PROTOCOL_PREFIX + addr;
                    })
                    .collect(Collectors.joining(","));
        }
        return address;
    }
    @Override
    public String setConfig(String key, String value) {
        return setConfig(null, key, value);
    }

    @Override
    public String getConfig(String key) {
        return getConfig(null, key);
    }

    @Override
    public boolean deleteConfig(String key) {
        return deleteConfig(null, key);
    }

    @Override
    public String setConfig(String group, String key, String value) {
        if (group == null) {
            group = namespace;
        }
        OpenItemDTO openItemDTO = new OpenItemDTO();
        openItemDTO.setKey(key);
        openItemDTO.setValue(value);
        client.createItem(appId, env, cluster, group, openItemDTO);
        return value;
    }

    @Override
    public String getConfig(String group, String key) {
        if (group == null) {
            group = namespace;
        }
        OpenItemDTO openItemDTO =  client.getItem(appId, env, cluster, group, key);
        if (openItemDTO != null) {
            return openItemDTO.getValue();
        }
        return null;
    }

    @Override
    public boolean deleteConfig(String group, String key) {
        if (group == null) {
            group = namespace;
        }
        //TODO user login user name as the operator
        client.removeItem(appId, env, cluster, group, key, "admin");
        return true;
    }

    @Override
    public String getPath(String key) {
        return null;
    }

    @Override
    public String getPath(String group, String key) {
        return null;
    }
}
