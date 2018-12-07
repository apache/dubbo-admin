package org.apache.dubbo.admin.service.impl;

import org.apache.dubbo.config.ApplicationConfig;
import org.apache.dubbo.config.ReferenceConfig;
import org.apache.dubbo.config.RegistryConfig;
import org.apache.dubbo.registry.Registry;
import org.apache.dubbo.rpc.service.GenericService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import javax.annotation.PostConstruct;

@Component
public class GenericServiceImpl {

    private ReferenceConfig<GenericService> reference;

    @Autowired
    private Registry registry;

    @PostConstruct
    public void init() {
        reference = new ReferenceConfig<>();
        reference.setGeneric(true);

        RegistryConfig registryConfig = new RegistryConfig();
        registryConfig.setAddress(registry.getUrl().getProtocol() + "://" + registry.getUrl().getAddress());

        ApplicationConfig applicationConfig = new ApplicationConfig();
        applicationConfig.setName("dubbo-admin");
        applicationConfig.setRegistry(registryConfig);

        reference.setApplication(applicationConfig);
    }

    public Object invoke(String service, String method, String[] parameterTypes, Object[] params) {

        reference.setInterface(service);
        GenericService genericService = reference.get();
        return genericService.$invoke(method, parameterTypes, params);
    }
}
