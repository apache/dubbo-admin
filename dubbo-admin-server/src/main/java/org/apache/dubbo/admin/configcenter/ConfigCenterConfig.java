package org.apache.dubbo.admin.configcenter;

import static org.apache.dubbo.common.config.ConfigurationUtils.parseProperties;

import java.io.IOException;

import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.config.Environment;
import org.apache.dubbo.common.extension.ExtensionLoader;
import org.apache.dubbo.configcenter.DynamicConfiguration;
import org.apache.dubbo.configcenter.DynamicConfigurationFactory;
import org.springframework.context.annotation.Bean;

public class ConfigCenterConfig extends org.apache.dubbo.config.ConfigCenterConfig {

    private static final long serialVersionUID = 7920845503724712940L;

    @Bean
    public DynamicConfiguration getDynamicConfiguration() {
        
        DynamicConfiguration dynamicConfiguration = getDynamicConfiguration(super.toUrl());

        String configContent = dynamicConfiguration.getProperties(super.getConfigFile(), super.getGroup());
        
        updateEnvironment(configContent);
        
        dynamicConfiguration.addListener(super.getConfigFile(), event -> {
            updateEnvironment(event.getValue());
        });
        
        return dynamicConfiguration;

    }
    
    
    private void updateEnvironment(String configContent) {
        try {
            Environment.getInstance().setConfigCenterFirst(super.isHighestPriority());
            Environment.getInstance().updateExternalConfigurationMap(parseProperties(configContent));
        } catch (IOException e) {
            throw new IllegalStateException("Failed to parse configurations from Config Center.", e);
        }
        
    }

    private DynamicConfiguration getDynamicConfiguration(URL url) {
        DynamicConfigurationFactory factory = ExtensionLoader
                .getExtensionLoader(DynamicConfigurationFactory.class)
                .getExtension(url.getProtocol());
        DynamicConfiguration configuration = factory.getDynamicConfiguration(url);
        Environment.getInstance().setDynamicConfiguration(configuration);
        return configuration;
    }

}
