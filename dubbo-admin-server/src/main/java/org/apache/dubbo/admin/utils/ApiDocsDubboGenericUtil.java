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
package org.apache.dubbo.admin.utils;

import org.apache.dubbo.config.ApplicationConfig;
import org.apache.dubbo.config.ReferenceConfig;
import org.apache.dubbo.config.RegistryConfig;
import org.apache.dubbo.rpc.service.GenericService;

import org.apache.commons.lang3.StringUtils;
import org.apache.commons.lang3.concurrent.BasicThreadFactory;

import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledThreadPoolExecutor;

/**
 * Dubbo operation related tool class.
 */
public class ApiDocsDubboGenericUtil {

    /**
     * Current application information.
     */
    private static ApplicationConfig application;

    /**
     * Registry information cache.
     */
    private static Map<String, RegistryConfig> registryConfigCache;

    /**
     * Dubbo service interface proxy cache.
     */
    private static Map<String, ReferenceConfig<GenericService>> referenceCache;

    private static final ScheduledExecutorService EXECUTOR;

    /**
     * Default retries.
     */
    private static int retries = 2;

    /**
     * Default timeout.
     */
    private static int timeout = 1000;

    static {
        // T (number of threads) = N (number of server cores) * u (expected CPU utilization) * (1 + E (waiting time) / C (calculation time))
        EXECUTOR = new ScheduledThreadPoolExecutor(
                Runtime.getRuntime().availableProcessors() * 40 * (1 + 5 / 2),
                new BasicThreadFactory.Builder().namingPattern("dubbo-async-executor-pool-%d").daemon(true).build());
        application = new ApplicationConfig();
        application.setName("dubbo-admin-api-docs");
        registryConfigCache = new ConcurrentHashMap<>();
        referenceCache = new ConcurrentHashMap<>();
    }

    public static void setRetriesAndTimeout(int retries, int timeout) {
        ApiDocsDubboGenericUtil.retries = retries;
        ApiDocsDubboGenericUtil.timeout = timeout;
    }

    /**
     * Get registry information.
     *
     * @param address Address of Registration Center
     * @return org.apache.dubbo.config.RegistryConfig
     */
    private static RegistryConfig getRegistryConfig(String address) {
        RegistryConfig registryConfig = registryConfigCache.get(address);
        if (null == registryConfig) {
            registryConfig = new RegistryConfig();
            registryConfig.setAddress(address);
            registryConfig.setRegister(false);
            registryConfigCache.put(address, registryConfig);
        }
        return registryConfig;
    }

    /**
     * remove cached registry information.
     * @param address Address of Registration Center
     * @return void
     */
    private static void removeRegistryConfig(String address) {
        registryConfigCache.remove(address);
    }

    /**
     * Get proxy object for dubbo service
     *
     * @return org.apache.dubbo.config.ReferenceConfig<org.apache.dubbo.rpc.service.GenericService>
     * @param: address  address Address of Registration Center
     * @param: interfaceName  Interface full package path
     */
    private static ReferenceConfig<GenericService> getReferenceConfig(String address, String interfaceName, String version) {
        final String key = buildCacheKey(address, interfaceName, version);
        ReferenceConfig<GenericService> referenceConfig = referenceCache.get(key);
        if (null == referenceConfig) {
            referenceConfig = new ReferenceConfig<>();
            referenceConfig.setRetries(retries);
            referenceConfig.setTimeout(timeout);
            referenceConfig.setVersion(version);
            referenceConfig.setApplication(application);
            if (address.startsWith("dubbo")) {
                referenceConfig.setUrl(address);
            } else {
                referenceConfig.setRegistry(getRegistryConfig(address));
            }
            referenceConfig.setInterface(interfaceName);
            // Declared as a generic interface
            referenceConfig.setGeneric(true);
            referenceCache.put(key, referenceConfig);
        }
        return referenceConfig;
    }

    /**
     * remove cached proxy object.
     * @param address
     * @param interfaceName
     * @return void
     */
    private static void removeReferenceConfig(String address, String interfaceName, String version) {
        removeRegistryConfig(address);
        referenceCache.remove(buildCacheKey(address, interfaceName, version));
    }

    private static String buildCacheKey(String address, String interfaceName, String version) {
        return address + "/" + interfaceName + "/" + version;
    }

    /**
     * Call duboo provider and return {@link CompletableFuture}
     *
     * @return java.util.concurrent.CompletableFuture<java.lang.Object>
     * @param: address
     * @param: interfaceName
     * @param: methodName
     * @param: async  Whether the provider is asynchronous is to directly return the {@link CompletableFuture}
     * returned by the provider, not to wrap it as {@link CompletableFuture}
     * @param: paramTypes
     * @param: paramValues
     */
    public static CompletableFuture<Object> invoke(String address, String interfaceName,
                                                   String methodName, boolean async, String version, String[] paramTypes,
                                                   Object[] paramValues) {
        CompletableFuture future = null;
        ReferenceConfig<GenericService> reference = getReferenceConfig(address, interfaceName, version);
        if (null != reference) {
            GenericService genericService = reference.get();
            if (null != genericService) {
                if (async) {
                    future = genericService.$invokeAsync(methodName, paramTypes, paramValues);
                } else {
                    future = CompletableFuture.supplyAsync(() -> genericService.$invoke(methodName, paramTypes, paramValues), EXECUTOR);
                }
            }
            future.exceptionally(ex -> {
                if (StringUtils.contains(ex.toString(), "Failed to invoke remote method")) {
                    removeReferenceConfig(address, interfaceName, version);
                }
                return ex;
            });
        }
        return future;
    }

    /**
     * Synchronous call provider. The provider must provide synchronous api
     * 2020/11/2 11:03
     *
     * @return java.lang.Object
     * @param: address
     * @param: interfaceName
     * @param: methodName
     * @param: paramTypes
     * @param: paramValues
     */
    public static Object invokeSync(String address, String interfaceName,
                                    String methodName, String version, String[] paramTypes,
                                    Object[] paramValues) {
        ReferenceConfig<GenericService> reference = getReferenceConfig(address, interfaceName, version);
        if (null != reference) {
            GenericService genericService = reference.get();
            try {
                if (null != genericService) {
                    return genericService.$invoke(methodName, paramTypes, paramValues);
                }
            } catch (Exception ex) {
                if (StringUtils.contains(ex.toString(), "Failed to invoke remote method")) {
                    removeReferenceConfig(address, interfaceName, version);
                } else {
                    throw ex;
                }
            }
        }
        return null;
    }

}
