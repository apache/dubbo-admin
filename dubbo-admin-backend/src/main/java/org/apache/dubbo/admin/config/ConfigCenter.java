package org.apache.dubbo.admin.config;

import org.apache.commons.lang3.StringUtils;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.data.config.GovernanceConfiguration;
import org.apache.dubbo.admin.data.metadata.MetaDataCollector;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.extension.ExtensionLoader;
import org.apache.dubbo.registry.Registry;
import org.apache.dubbo.registry.RegistryFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.DependsOn;

import java.util.Arrays;


@Configuration
public class ConfigCenter {


    @Value("${dubbo.configcenter:}")
    private String configCenter;

    @Value("${dubbo.registry.address:}")
    private String registryAddress;

    private static String globalConfigPath = "/config/dubbo/dubbo.properties";

    @Value("${dubbo.registry.group:}")
    private String group;


    private URL configCenterUrl;
    private URL registryUrl;
    private URL metadataUrl;


    /*
     * generate dynamic configuration client
     */
    @Bean("governanceConfiguration")
    GovernanceConfiguration getDynamicConfiguration() {
        if (configCenter != null) {
            configCenterUrl = formUrl(configCenter, group);
            GovernanceConfiguration dynamicConfiguration = ExtensionLoader.getExtensionLoader(GovernanceConfiguration.class).getExtension(configCenterUrl.getProtocol());
            dynamicConfiguration.setUrl(configCenterUrl);
            dynamicConfiguration.init();
            globalConfigPath = group == null ? "/dubbo" + globalConfigPath : "/" + group + globalConfigPath;
            String config = dynamicConfiguration.getConfig(globalConfigPath);

            Arrays.stream(config.split("\n")).forEach( s -> {
                if(s.startsWith(Constants.REGISTRY_ADDRESS)) {
                    registryUrl = formUrl(s.split("=")[1].trim(), group);
                } else if (s.startsWith(Constants.METADATA_ADDRESS)) {
                    metadataUrl = formUrl(s.split("=")[1].trim(), group);
                }
            });
            return dynamicConfiguration;
        }
        return null;
    }

    /*
     * generate registry client
     */
    @Bean
    @DependsOn("governanceConfiguration")
    Registry getRegistry() {
        Registry registry = null;
        if (registryUrl != null) {
            RegistryFactory registryFactory = ExtensionLoader.getExtensionLoader(RegistryFactory.class).getAdaptiveExtension();
            registry = registryFactory.getRegistry(registryUrl);
        }
        return registry;
    }

    /*
     * generate metadata client
     */
    @Bean
    @DependsOn("governanceConfiguration")
    MetaDataCollector getMetadataCollector() {
        MetaDataCollector metaDataCollector = null;
        if (metadataUrl != null) {
            metaDataCollector = ExtensionLoader.getExtensionLoader(MetaDataCollector.class).getExtension(metadataUrl.getProtocol());
            metaDataCollector.setUrl(metadataUrl);
            metaDataCollector.init();
        }
        return metaDataCollector;
    }

    private URL formUrl(String config, String group) {
        String protocol = config.split("://")[0];
        String address = config.split("://")[1];
        String port = address.split(":")[1];
        String host = address.split(":")[0];
        URL url = new URL(protocol, host, Integer.parseInt(port));
        if (StringUtils.isNotEmpty(group)) {
            url.addParameter(org.apache.dubbo.common.Constants.GROUP_KEY, group);
        }
        return url;
    }
}
