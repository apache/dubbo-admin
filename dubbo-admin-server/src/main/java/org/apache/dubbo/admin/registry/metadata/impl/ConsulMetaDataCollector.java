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

package org.apache.dubbo.admin.registry.metadata.impl;


import org.apache.dubbo.admin.registry.metadata.MetaDataCollector;

import com.ecwid.consul.v1.ConsulClient;
import com.ecwid.consul.v1.Response;
import com.ecwid.consul.v1.kv.model.GetValue;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.logger.Logger;
import org.apache.dubbo.common.logger.LoggerFactory;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;

import java.util.Objects;


public class ConsulMetaDataCollector implements MetaDataCollector {
    private static final Logger LOG = LoggerFactory.getLogger(ConsulMetaDataCollector.class);
    private static final int DEFAULT_PORT = 8500;
    private URL url;
    private ConsulClient client;

    @Override
    public URL getUrl() {
        return this.url;
    }

    @Override
    public void setUrl(URL url) {
        this.url = url;
    }

    @Override
    public void init() {
        Objects.requireNonNull(this.url, "metadataUrl require not null");
        String host = this.url.getHost();
        int port = this.url.getPort() != 0 ? url.getPort() : DEFAULT_PORT;
        this.client = new ConsulClient(host, port);
    }

    @Override
    public String getProviderMetaData(MetadataIdentifier key) {
        return doGetMetaData(key);
    }

    @Override
    public String getConsumerMetaData(MetadataIdentifier key) {
        return doGetMetaData(key);
    }

    private String doGetMetaData(MetadataIdentifier key) {
        try {
            Response<GetValue> response = this.client.getKVValue(key.getUniqueKey(MetadataIdentifier.KeyTypeEnum.UNIQUE_KEY));
            return response.getValue().getDecodedValue();
        } catch (Exception e) {
            LOG.error(String.format("Failed to fetch metadata for %s from consul, cause: %s",
                    key.getUniqueKey(MetadataIdentifier.KeyTypeEnum.UNIQUE_KEY), e.getMessage()), e);
        }
        return null;
    }

    //just for test
    ConsulClient getClient() {
        return this.client;
    }
}
