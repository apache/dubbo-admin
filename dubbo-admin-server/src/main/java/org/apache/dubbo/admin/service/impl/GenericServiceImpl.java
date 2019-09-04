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

package org.apache.dubbo.admin.service.impl;

import org.apache.dubbo.admin.common.util.Tool;
import org.apache.dubbo.config.ApplicationConfig;
import org.apache.dubbo.config.ReferenceConfig;
import org.apache.dubbo.config.RegistryConfig;
import org.apache.dubbo.registry.Registry;
import org.apache.dubbo.rpc.service.GenericService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import javax.annotation.PostConstruct;
import java.util.Iterator;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ScheduledThreadPoolExecutor;
import java.util.concurrent.TimeUnit;

@Component
public class GenericServiceImpl {
    protected static final Logger logger = LoggerFactory.getLogger(GenericServiceImpl.class);


    private ApplicationConfig applicationConfig;
    private final Registry registry;
    private ConcurrentHashMap<String, GenericServiceInfo> serviceMap = new ConcurrentHashMap<>();
    private static int DESTROY_TIMEOUT = 600;
    private static ScheduledThreadPoolExecutor destroyExecutor = new ScheduledThreadPoolExecutor(1);


    public GenericServiceImpl(Registry registry) {
        this.registry = registry;
    }

    @PostConstruct
    public void init() {
        RegistryConfig registryConfig = new RegistryConfig();
        registryConfig.setAddress(registry.getUrl().getProtocol() + "://" + registry.getUrl().getAddress());
        registryConfig.setGroup(registry.getUrl().getParameter("group"));

        applicationConfig = new ApplicationConfig();
        applicationConfig.setName("dubbo-admin");
        applicationConfig.setRegistry(registryConfig);
        destroyExecutor.scheduleAtFixedRate(new ReferenceServiceCleanTask(), DESTROY_TIMEOUT, DESTROY_TIMEOUT, TimeUnit.SECONDS);
    }

    public Object invoke(String service, String method, String[] parameterTypes, Object[] params) {

        GenericService genericService = getService(service);
        Object result = genericService.$invoke(method, parameterTypes, params);
        return result;
    }

    private GenericService getService(String service) {
        if (serviceMap.containsKey(service)) {
            GenericServiceInfo referenceInfo = serviceMap.get(service);
            referenceInfo.updateLastActiveTime();
            return referenceInfo.getReference().get();
        } else {
            ReferenceConfig<GenericService> reference = initReferenceConfig(service);
            return reference.get();
        }
    }

    private synchronized ReferenceConfig<GenericService> initReferenceConfig(String service) {
        if (serviceMap.containsKey(service)) {
            GenericServiceInfo referenceInfo = serviceMap.get(service);
            return referenceInfo.getReference();
        }
        ReferenceConfig<GenericService> reference = new ReferenceConfig<>();
        String group = Tool.getGroup(service);
        String version = Tool.getVersion(service);
        String interfaze = Tool.getInterface(service);
        reference.setGeneric(true);
        reference.setApplication(applicationConfig);
        reference.setInterface(interfaze);
        reference.setVersion(version);
        reference.setGroup(group);

        GenericServiceInfo referenceInfo = new GenericServiceInfo();
        referenceInfo.setReference(reference);
        serviceMap.put(service, referenceInfo);
        return reference;
    }

    class GenericServiceInfo {
        private ReferenceConfig<GenericService> reference;
        private long lastActiveTime = System.currentTimeMillis();

        public ReferenceConfig<GenericService> getReference() {
            return reference;
        }

        public void setReference(ReferenceConfig<GenericService> reference) {
            this.reference = reference;
        }

        long getLastActiveTime() {
            return lastActiveTime;
        }

        void updateLastActiveTime() {
            this.lastActiveTime = System.currentTimeMillis();
        }
    }

    class ReferenceServiceCleanTask implements Runnable {
        private long EXPIRY_TIMEOUT = 1800000;

        @Override
        public void run() {
            Iterator<Map.Entry<String, GenericServiceInfo>> iterator = serviceMap.entrySet().iterator();
            while (iterator.hasNext()) {
                Map.Entry<String, GenericServiceInfo> item = iterator.next();
                GenericServiceInfo service = item.getValue();
                if (checkExpiry(service.getLastActiveTime())) {
                    logger.info(String.format("test service %s expiry,will destroy", item.getKey()));
                    iterator.remove();
                    service.getReference().destroy();
                }
            }
        }

        private boolean checkExpiry(long activeTime) {
            return (System.currentTimeMillis() - activeTime) > EXPIRY_TIMEOUT;
        }
    }
}
