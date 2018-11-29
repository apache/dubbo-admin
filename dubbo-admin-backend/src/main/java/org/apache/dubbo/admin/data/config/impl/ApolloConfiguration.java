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
