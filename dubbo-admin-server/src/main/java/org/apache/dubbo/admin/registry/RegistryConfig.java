package org.apache.dubbo.admin.registry;

import org.apache.commons.lang3.StringUtils;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.extension.ExtensionLoader;
import org.apache.dubbo.configcenter.DynamicConfiguration;
import org.apache.dubbo.registry.Registry;
import org.apache.dubbo.registry.RegistryFactory;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;

public class RegistryConfig extends org.apache.dubbo.config.RegistryConfig {

    private static final long serialVersionUID = 8627088723517642419L;


    @Bean
    @ConditionalOnMissingBean(DynamicConfiguration.class)
    public Registry registry() {
        URL registryUrl = formUrl();
        RegistryFactory factory = ExtensionLoader.getExtensionLoader(RegistryFactory.class)
                .getExtension(registryUrl.getProtocol());
        return factory.getRegistry(registryUrl);
    }
    
    @Bean
    @ConditionalOnMissingBean
    public Registry configCenterRegistry(DynamicConfiguration config) {
        super.refresh();
        return registry();
    }

    private URL formUrl() {
        URL url = URL.valueOf(getAddress());
        if (StringUtils.isNotEmpty(getGroup())) {
            url = url.addParameter(Constants.GROUP_KEY, getGroup());
        }
        if (StringUtils.isNotEmpty(getUsername())) {
            url = url.setUsername(getUsername());
        }
        if (StringUtils.isNotEmpty(getPassword())) {
            url = url.setPassword(getPassword());
        }
        if (getParameters() != null ) {
            url = url.addParameters(getParameters());
        }
        return url;
    }
}
