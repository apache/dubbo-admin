package org.apache.dubbo.admin.registry.config;

import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.extension.ExtensionLoader;
import org.springframework.context.annotation.Bean;

public class GovernanceConfigurationConfig extends org.apache.dubbo.config.ConfigCenterConfig {

    private static final long serialVersionUID = -7036219290017247288L;
    
    @Bean
    public GovernanceConfiguration getGovernanceConfiguration() {
        URL configCenterUrl = super.toUrl();
        GovernanceConfiguration dynamicConfiguration = ExtensionLoader.getExtensionLoader(GovernanceConfiguration.class)
                .getExtension(configCenterUrl.getProtocol());
        dynamicConfiguration.setUrl(configCenterUrl);
        dynamicConfiguration.init();
        return dynamicConfiguration;
    }

}
