package org.apache.dubbo.admin.config;

import org.apache.dubbo.admin.data.config.GovernanceConfiguration;
import org.apache.dubbo.common.Constants;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.extension.ExtensionLoader;
import org.apache.dubbo.registry.Registry;
import org.apache.dubbo.registry.RegistryFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class ConfigCenter {


    @Value("${dubbo.configcenter}")
    private String configCenter;

    @Value("${dubbo.registry.address}")
    private String registryAddress;

    @Value("${dubbo.registry.group}")
    private String group;


    private URL configCenterUrl;
    private URL registryUrl;


    @Bean
    Registry getRegistryService() {
        if (registryAddress != null) {
            registryUrl = formUrl(registryAddress);
            if (group != null) {
                registryUrl.addParameter(Constants.GROUP_KEY, group);
            }
            RegistryFactory factory = ExtensionLoader.getExtensionLoader(RegistryFactory.class).getAdaptiveExtension();
            return factory.getRegistry(registryUrl);
        }
        return null;
    }

    @Bean
    GovernanceConfiguration getDynamicConfiguration() {
        if (configCenter != null) {
            configCenterUrl = formUrl(configCenter);
            GovernanceConfiguration dynamicConfiguration = ExtensionLoader.getExtensionLoader(GovernanceConfiguration.class).getExtension(configCenterUrl.getProtocol());
            dynamicConfiguration.setUrl(configCenterUrl);
            dynamicConfiguration.init();
            return dynamicConfiguration;
        }
        return null;
    }

    private URL formUrl(String config) {
        String protocol = config.split("://")[0];
        String address = config.split("://")[1];
        String port = address.split(":")[1];
        String host = address.split(":")[0];
        return new URL(protocol, host, Integer.parseInt(port));
    }

}
