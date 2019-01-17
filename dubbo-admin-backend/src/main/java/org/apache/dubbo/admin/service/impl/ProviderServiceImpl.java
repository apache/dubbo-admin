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
import org.apache.dubbo.admin.common.util.Pair;
import org.apache.dubbo.admin.common.util.ParseUtils;
import org.apache.dubbo.admin.common.util.SyncUtils;
import org.apache.dubbo.admin.model.domain.Provider;
import org.apache.dubbo.admin.model.dto.ServiceDTO;
import org.apache.dubbo.admin.service.OverrideService;
import org.apache.dubbo.admin.service.ProviderService;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.utils.StringUtils;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import java.util.*;
import java.util.Map.Entry;
import java.util.concurrent.ConcurrentMap;
import java.util.regex.Matcher;
import java.util.regex.Pattern;


/**
 * IbatisProviderService
 *
 */
@Component
public class ProviderServiceImpl extends AbstractService implements ProviderService {

    @Autowired
    OverrideService overrideService;

    @Override
    public void create(Provider provider) {
        URL url = provider.toUrl();
        registry.register(url);
    }

//    public void enableProvider(String id) {
//        if (id == null) {
//            throw new IllegalStateException("no provider id");
//        }
//
//        Provider oldProvider = findProvider(id);
//
//        if (oldProvider == null) {
//            throw new IllegalStateException("Provider was changed!");
//        }
//        if (oldProvider.isDynamic()) {
//            // Make sure we only have one override configured disable property.
//            if (!oldProvider.isEnabled()) {
//                Override override = new Override();
//                override.setAddress(oldProvider.getAddress());
//                override.setService(oldProvider.getService());
//                override.setEnabled(true);
//                override.setParams(Constants.DISABLED_KEY + "=false");
//                overrideService.saveOverride(override);
//                return;
//            }
//            List<Override> oList = overrideService.findByServiceAndAddress(oldProvider.getService(), oldProvider.getAddress());
//
//            for (Override o : oList) {
//                Map<String, String> params = StringUtils.parseQueryString(o.getParams());
//                if (params.containsKey(Constants.DISABLED_KEY)) {
//                    if (params.get(Constants.DISABLED_KEY).equals("true")) {
//                        overrideService.deleteOverride(o.getHash());
//                    }
//                }
//            }
//        } else {
//            oldProvider.setEnabled(true);
//            updateProvider(oldProvider);
//        }
//    }

    @Override
    public String getProviderMetaData(MetadataIdentifier providerIdentifier) {
        return metaDataCollector.getProviderMetaData(providerIdentifier);
    }

//    public void disableProvider(String id) {
//        if (id == null) {
//            throw new IllegalStateException("no provider id");
//        }
//
//        Provider oldProvider = findProvider(id);
//        if (oldProvider == null) {
//            throw new IllegalStateException("Provider was changed!");
//        }
//
//        if (oldProvider.isDynamic()) {
//            // Make sure we only have one override configured disable property.
//            if (oldProvider.isEnabled()) {
//                Override override = new Override();
//                override.setAddress(oldProvider.getAddress());
//                override.setService(oldProvider.getService());
//                override.setEnabled(true);
//                override.setParams(Constants.DISABLED_KEY + "=true");
//                overrideService.saveOverride(override);
//                return;
//            }
//            List<Override> oList = overrideService.findByServiceAndAddress(oldProvider.getService(), oldProvider.getAddress());
//
//            for (Override o : oList) {
//                Map<String, String> params = StringUtils.parseQueryString(o.getParams());
//                if (params.containsKey(Constants.DISABLED_KEY)) {
//                    if (params.get(Constants.DISABLED_KEY).equals("false")) {
//                        overrideService.deleteOverride(o.getHash());
//                    }
//                }
//            }
//        } else {
//            oldProvider.setEnabled(false);
//            updateProvider(oldProvider);
//        }
//
//    }

//    public void doublingProvider(String id) {
//        setWeight(id, 2F);
//    }
//
//    public void halvingProvider(String id) {
//        setWeight(id, 0.5F);
//    }

//    public void setWeight(String id, float factor) {
//        if (id == null) {
//            throw new IllegalStateException("no provider id");
//        }
//        Provider oldProvider = findProvider(id);
//        if (oldProvider == null) {
//            throw new IllegalStateException("Provider was changed!");
//        }
//        Map<String, String> map = StringUtils.parseQueryString(oldProvider.getParameters());
//        String weight = map.get(Constants.WEIGHT_KEY);
//        if (oldProvider.isDynamic()) {
//            // Make sure we only have one override configured disable property.
//            List<Override> overrides = overrideService.findByServiceAndAddress(oldProvider.getService(), oldProvider.getAddress());
//            if (overrides == null || overrides.size() == 0) {
//                int value = getWeight(weight, factor);
//                if (value != Constants.DEFAULT_WEIGHT) {
//                    Override override = new Override();
//                    override.setAddress(oldProvider.getAddress());
//                    override.setService(oldProvider.getService());
//                    override.setEnabled(true);
//                    override.setParams(Constants.WEIGHT_KEY + "=" + String.valueOf(value));
//                    overrideService.saveOverride(override);
//                }
//            } else {
//                for (Override override : overrides) {
//                    Map<String, String> params = StringUtils.parseQueryString(override.getParams());
//                    String overrideWeight = params.get(Constants.WEIGHT_KEY);
//                    if (overrideWeight == null || overrideWeight.length() == 0) {
//                        overrideWeight = weight;
//                    }
//                    int value = getWeight(overrideWeight, factor);
//                    if (value == getWeight(weight, 1)) {
//                        params.remove(Constants.WEIGHT_KEY);
//                    } else {
//                        params.put(Constants.WEIGHT_KEY, String.valueOf(value));
//                    }
//                    if (params.size() > 0) {
//                        override.setParams(StringUtils.toQueryString(params));
//                        overrideService.updateOverride(override);
//                    } else {
//                        overrideService.deleteOverride(override.getHash());
//                    }
//                }
//            }
//        } else {
//            int value = getWeight(weight, factor);
//            if (value == Constants.DEFAULT_WEIGHT) {
//                map.remove(Constants.WEIGHT_KEY);
//            } else {
//                map.put(Constants.WEIGHT_KEY, String.valueOf(value));
//            }
//            oldProvider.setParameters(StringUtils.toQueryString(map));
//            updateProvider(oldProvider);
//        }
//    }

    private int getWeight(String value, float factor) {
        int weight = 100;
        if (value != null && value.length() > 0) {
            weight = Integer.parseInt(value);
        }
        weight = (int) (weight * factor);
        if (weight < 1){
            weight = 1;
        }
        if (weight == 2){
            weight = 3;
        }

        if (weight == 24){
            weight = 25;
        }
        return weight;
    }

    @Override
    public void deleteStaticProvider(String id) {
        URL oldProvider = findProviderUrl(id);
        if (oldProvider == null) {
            throw new IllegalStateException("Provider was changed!");
        }
        registry.unregister(oldProvider);
    }

    @Override
    public void updateProvider(Provider provider) {
        String hash = provider.getHash();
        if (hash == null) {
            throw new IllegalStateException("no provider id");
        }

        URL oldProvider = findProviderUrl(hash);
        if (oldProvider == null) {
            throw new IllegalStateException("Provider was changed!");
        }
        URL newProvider = provider.toUrl();

        registry.unregister(oldProvider);
        registry.register(newProvider);
    }

    @Override
    public Provider findProvider(String id) {
        return SyncUtils.url2Provider(findProviderUrlPair(id));
    }

    public Pair<String, URL> findProviderUrlPair(String id) {
        return SyncUtils.filterFromCategory(getRegistryCache(), Constants.PROVIDERS_CATEGORY, id);
    }

    @Override
    public Set<String> findServices() {
        Set<String> ret = new HashSet<>();
        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (providerUrls != null){
            ret.addAll(providerUrls.keySet());
        }
        return ret;
    }

    @Override
    public List<String> findAddresses() {
        List<String> ret = new ArrayList<String>();

        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
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
    public List<String> findAddressesByApplication(String application) {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        for (Map.Entry<String, Map<String, URL>> e1 : providerUrls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                if (application.equals(u.getParameter(Constants.APPLICATION))) {
                    String addr = u.getAddress();
                    if (addr != null) {
                        ret.add(addr);
                    }
                }
            }
        }

        return ret;
    }

    @Override
    public List<String> findAddressesByService(String service) {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (null == providerUrls) {
            return ret;
        }

        for (Map.Entry<String, URL> e2 : providerUrls.get(service).entrySet()) {
            URL u = e2.getValue();
            String app = u.getAddress();
            if (app != null) {
                ret.add(app);
            }
        }

        return ret;
    }

    @Override
    public List<String> findApplicationsByServiceName(String service) {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (null == providerUrls) {
            return ret;
        }

        Map<String, URL> value = providerUrls.get(service);
        if (value == null) {
            return ret;
        }
        for (Map.Entry<String, URL> e2 : value.entrySet()) {
            URL u = e2.getValue();
            String app = u.getParameter(Constants.APPLICATION);
            if (app != null){
                ret.add(app);
            }
        }

        return ret;
    }

    @Override
    public List<Provider> findByService(String serviceName) {
        return SyncUtils.url2ProviderList(findProviderUrlByService(serviceName));
    }

    @Override
    public List<Provider> findByAppandService(String app, String serviceName) {
        return SyncUtils.url2ProviderList(findProviderUrlByAppandService(app, serviceName));
    }

    private Map<String, URL> findProviderUrlByService(String service) {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY);
        filter.put(SyncUtils.SERVICE_FILTER_KEY, service);

        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    @Override
    public List<Provider> findAll() {
        return SyncUtils.url2ProviderList(findAllProviderUrl());
    }

    private Map<String, URL> findAllProviderUrl() {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY);
        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    @Override
    public List<Provider> findByAddress(String providerAddress) {
        return SyncUtils.url2ProviderList(findProviderUrlByAddress(providerAddress));
    }

    public Map<String, URL> findProviderUrlByAddress(String address) {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY);
        filter.put(SyncUtils.ADDRESS_FILTER_KEY, address);

        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    @Override
    public List<String> findServicesByAddress(String address) {
        List<String> ret = new ArrayList<String>();

        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (providerUrls == null || address == null || address.length() == 0) {
            return ret;
        }

        for (Map.Entry<String, Map<String, URL>> e1 : providerUrls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                if (address.equals(u.getAddress())) {
                    ret.add(e1.getKey());
                    break;
                }
            }
        }

        return ret;
    }

    @Override
    public Set<String> findApplications() {
        Set<String> ret = new HashSet<>();
        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (providerUrls == null){
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
        return SyncUtils.url2ProviderList(findProviderUrlByApplication(application));
    }

    @Override
    public String findVersionInApplication(String application) {
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
        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }



    private Map<String, URL> findProviderUrlByApplication(String application) {
        Map<String, String> filter = new HashMap<>();
        filter.put(Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY);
        filter.put(Constants.APPLICATION, application);
        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    @Override
    public List<String> findServicesByApplication(String application) {
        List<String> ret = new ArrayList<String>();

        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
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
    public List<String> findMethodsByService(String service) {
        List<String> ret = new ArrayList<String>();

        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (providerUrls == null || service == null || service.length() == 0){
            return ret;
        }

        Map<String, URL> providers = providerUrls.get(service);
        if (null == providers || providers.isEmpty()) {
            return ret;
        }

        Entry<String, URL> p = providers.entrySet().iterator().next();
        String value = p.getValue().getParameter("methods");
        if (value == null || value.length() == 0) {
            return ret;
        }
        String[] methods = value.split(ParseUtils.METHOD_SPLIT);
        if (methods == null || methods.length == 0) {
            return ret;
        }

        for (String m : methods) {
            ret.add(m);
        }
        return ret;
    }

    private URL findProviderUrl(String id) {
        return findProvider(id).toUrl();
    }

    @Override
    public Provider findByServiceAndAddress(String service, String address) {
        return SyncUtils.url2Provider(findProviderUrl(service, address));
    }

    private Pair<String, URL> findProviderUrl(String service, String address) {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY);
        filter.put(SyncUtils.ADDRESS_FILTER_KEY, address);

        Map<String, URL> ret = SyncUtils.filterFromCategory(getRegistryCache(), filter);
        if (ret.isEmpty()) {
            return null;
        } else {
            String key = ret.entrySet().iterator().next().getKey();
            return new Pair<String, URL>(key, ret.get(key));
        }
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
                    } else {
                        providers.addAll(findByApplication(candidate));
                    }
                }
            }
        }

        Set<ServiceDTO> result = convertProviders2DTO(providers);
        return result;
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
            Map<String, String> map = StringUtils.parseQueryString(provider.getParameters());
            String app = provider.getApplication();
            String service = map.get(Constants.INTERFACE_KEY);
            String group = map.get(Constants.GROUP_KEY);
            String version = map.get(Constants.VERSION_KEY);
            ServiceDTO s = new ServiceDTO();
            s.setAppName(app);
            s.setService(service);
            s.setGroup(group);
            s.setVersion(version);
            result.add(s);
        }
        return result;
    }

}
