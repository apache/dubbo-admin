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

import org.apache.dubbo.admin.common.exception.ParamValidationException;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.common.util.SyncUtils;
import org.apache.dubbo.admin.common.util.Tool;
import org.apache.dubbo.admin.model.domain.Provider;
import org.apache.dubbo.admin.model.dto.ServiceDTO;
import org.apache.dubbo.admin.service.ProviderService;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.metadata.report.identifier.MetadataIdentifier;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.TreeSet;
import java.util.concurrent.ConcurrentMap;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Collectors;

@Component
public class ProviderServiceImpl extends AbstractService implements ProviderService {

    @Autowired
    private InstanceRegistryQueryHelper instanceRegistryQueryHelper;

    @Override
    public void create(Provider provider) {
        URL url = provider.toUrl();
        registry.register(url);
    }


    @Override
    public String getProviderMetaData(MetadataIdentifier providerIdentifier) {
        return metaDataCollector.getProviderMetaData(providerIdentifier);
    }

    @Override
    public Set<String> findServices() {
        Set<String> ret = new HashSet<>();
        ConcurrentMap<String, Map<String, URL>> providerUrls = getInterfaceRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (providerUrls != null) {
            ret.addAll(providerUrls.keySet());
        }
        ret.addAll(instanceRegistryQueryHelper.findServices());
        return ret;
    }

    @Override
    public Set<String> findInstanceApplications() {
        return instanceRegistryQueryHelper.findApplications();
    }

    @Override
    public List<String> findAddresses() {
        List<String> ret = new ArrayList<String>();

        ConcurrentMap<String, Map<String, URL>> providerUrls = getInterfaceRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (null == providerUrls) {
            return ret;
        }

        for (Map.Entry<String, Map<String, URL>> e1 : providerUrls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                String app = u.getAddress();
                if (app != null) {
                    ret.add(app);
                }
            }
        }
        return ret;
    }

    @Override
    public List<Provider> findByService(String serviceName) {
        List<Provider> instanceProviders = instanceRegistryQueryHelper.findByService(serviceName);
        List<Provider> interfaceProviders = SyncUtils.url2ProviderList(findProviderUrlByService(serviceName));
        instanceProviders.addAll(interfaceProviders);
        return instanceProviders;
    }

    private Map<String, URL> findProviderUrlByService(String service) {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY);
        filter.put(SyncUtils.SERVICE_FILTER_KEY, service);

        return SyncUtils.filterFromCategory(getInterfaceRegistryCache(), filter);
    }

    @Override
    public List<Provider> findAll() {
        return SyncUtils.url2ProviderList(findAllProviderUrl());
    }

    private Map<String, URL> findAllProviderUrl() {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY);
        return SyncUtils.filterFromCategory(getInterfaceRegistryCache(), filter);
    }

    @Override
    public List<Provider> findByAddress(String providerAddress) {
        List<Provider> instanceProviders = instanceRegistryQueryHelper.findByAddress(providerAddress);
        List<Provider> interfaceProviders = SyncUtils.url2ProviderList(findProviderUrlByAddress(providerAddress));
        instanceProviders.addAll(interfaceProviders);
        return instanceProviders;
    }

    public Map<String, URL> findProviderUrlByAddress(String address) {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY);
        filter.put(SyncUtils.ADDRESS_FILTER_KEY, address);

        return SyncUtils.filterFromCategory(getInterfaceRegistryCache(), filter);
    }

    @Override
    public Set<String> findApplications() {
        Set<String> ret = instanceRegistryQueryHelper.findApplications();
        ConcurrentMap<String, Map<String, URL>> providerUrls = getInterfaceRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (providerUrls == null) {
            return ret;
        }

        for (Map.Entry<String, Map<String, URL>> e1 : providerUrls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                String app = u.getParameter(Constants.APPLICATION);
                if (app != null) {
                    ret.add(app);
                }
            }
        }

        return ret;
    }

    @Override
    public List<Provider> findByApplication(String application) {
        List<Provider> instanceProviders = instanceRegistryQueryHelper.findByApplication(application);
        List<Provider> interfaceProviders = SyncUtils.url2ProviderList(findProviderUrlByApplication(application));
        instanceProviders.addAll(interfaceProviders);
        return instanceProviders;
    }

    @Override
    public String findVersionInApplication(String application) {
        String version = instanceRegistryQueryHelper.findVersionInApplication(application);
        if (StringUtils.isNotBlank(version)){
            return version;
        }
        List<String> services = findServicesByApplication(application);
        if (services == null || services.size() == 0) {
            throw new ParamValidationException("there is no service for application: " + application);
        }
        return findServiceVersion(services.get(0), application);
    }

    @Override
    public String findServiceVersion(String serviceName, String application) {
        String version = "2.6";
        Map<String, URL> result = findProviderUrlByAppandService(application, serviceName);
        if (result != null && result.size() > 0) {
            URL url = result.values().stream().findFirst().get();
            if (url.getParameter(Constants.SPECIFICATION_VERSION_KEY) != null) {
                version = url.getParameter(Constants.SPECIFICATION_VERSION_KEY);
            }
        }
        return version;
    }

    private Map<String, URL> findProviderUrlByAppandService(String app, String service) {
        Map<String, String> filter = new HashMap<>();
        filter.put(Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY);
        filter.put(Constants.APPLICATION, app);
        filter.put(SyncUtils.SERVICE_FILTER_KEY, service);
        return SyncUtils.filterFromCategory(getInterfaceRegistryCache(), filter);
    }


    private Map<String, URL> findProviderUrlByApplication(String application) {
        Map<String, String> filter = new HashMap<>();
        filter.put(Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY);
        filter.put(Constants.APPLICATION, application);
        return SyncUtils.filterFromCategory(getInterfaceRegistryCache(), filter);
    }

    @Override
    public List<String> findServicesByApplication(String application) {
        List<String> ret = new ArrayList<String>();

        ConcurrentMap<String, Map<String, URL>> providerUrls = getInterfaceRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (providerUrls == null || application == null || application.length() == 0) {
            return ret;
        }

        for (Map.Entry<String, Map<String, URL>> e1 : providerUrls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                if (application.equals(u.getParameter(Constants.APPLICATION))) {
                    ret.add(e1.getKey());
                    break;
                }
            }
        }

        return ret;
    }

    @Override
    public Set<ServiceDTO> getServiceDTOS(String pattern, String filter, String env) {
        List<Provider> providers = new ArrayList<>();
        if (!filter.contains(Constants.ANY_VALUE) && !filter.contains(Constants.INTERROGATION_POINT)) {
            // filter with specific string
            if (Constants.IP.equals(pattern)) {
                providers = findByAddress(filter);
            } else if (Constants.SERVICE.equals(pattern)) {
                providers = findByService(filter);
            } else if (Constants.APPLICATION.equals(pattern)) {
                providers = findByApplication(filter);
            }
        } else {
            // filter with fuzzy search
            Set<String> candidates = Collections.emptySet();
            if (Constants.SERVICE.equals(pattern)) {
                candidates = findServices();
            } else if (Constants.APPLICATION.equals(pattern)) {
                candidates = findApplications();
            } else if (Constants.IP.equals(pattern)) {
                candidates = findAddresses().stream().collect(Collectors.toSet());
            }
            // replace dot symbol and asterisk symbol to java-based regex pattern
            filter = filter.toLowerCase().replace(Constants.PUNCTUATION_POINT, Constants.PUNCTUATION_SEPARATOR_POINT);
            // filter start with [* 、? 、+] will triggering PatternSyntaxException
            if (filter.startsWith(Constants.ANY_VALUE)
                    || filter.startsWith(Constants.INTERROGATION_POINT) || filter.startsWith(Constants.PLUS_SIGNS)) {
                filter = Constants.PUNCTUATION_POINT + filter;
            }
            // search with no case insensitive
            Pattern regex = Pattern.compile(filter, Pattern.CASE_INSENSITIVE);
            for (String candidate : candidates) {
                Matcher matcher = regex.matcher(candidate);
                if (matcher.matches() || matcher.lookingAt()) {
                    if (Constants.SERVICE.equals(pattern)) {
                        providers.addAll(findByService(candidate));
                    } else if (Constants.IP.equals(pattern)) {
                        providers.addAll(findByAddress(candidate));
                    } else {
                        providers.addAll(findByApplication(candidate));
                    }
                }
            }
        }

        return convertProviders2DTO(providers);
    }

    /**
     * Convert provider list to ServiceDTO list
     *
     * @param providers list of providers
     * @return ServiceDTO list of front page
     */
    public Set<ServiceDTO> convertProviders2DTO(List<Provider> providers) {
        Set<ServiceDTO> result = new TreeSet<>();
        for (Provider provider : providers) {
            String app = provider.getApplication();
            String service = provider.getService();
            String group = Tool.getGroup(service);
            String version = Tool.getVersion(service);
            String interfaze = Tool.getInterface(service);
            ServiceDTO s = new ServiceDTO();
            s.setAppName(app);
            s.setService(interfaze);
            s.setGroup(group);
            s.setVersion(version);
            s.setRegistrySource(provider.getRegistrySource());
            result.add(s);
        }
        return result;
    }

}
