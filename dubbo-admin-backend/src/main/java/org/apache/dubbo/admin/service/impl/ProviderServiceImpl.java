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

import com.google.common.collect.Iterables;
import com.google.gson.Gson;
import org.apache.dubbo.admin.common.exception.ParamValidationException;
import org.apache.dubbo.admin.common.util.ConvertUtil;
import org.apache.dubbo.admin.common.util.Pair;
import org.apache.dubbo.admin.common.util.ParseUtils;
import org.apache.dubbo.admin.common.util.SyncUtils;
import org.apache.dubbo.admin.model.domain.Provider;
import org.apache.dubbo.admin.service.OverrideService;
import org.apache.dubbo.admin.service.ProviderService;
import org.apache.dubbo.common.Constants;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.utils.StringUtils;
import org.apache.dubbo.metadata.definition.model.FullServiceDefinition;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;
import org.yaml.snakeyaml.events.Event;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Map.Entry;
import java.util.concurrent.ConcurrentMap;

import static org.apache.dubbo.common.Constants.SPECIFICATION_VERSION_KEY;

/**
 * IbatisProviderService
 *
 */
@Component
public class ProviderServiceImpl extends AbstractService implements ProviderService {

    @Autowired
    OverrideService overrideService;

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

    @java.lang.Override
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
        if (weight < 1) weight = 1;
        if (weight == 2) weight = 3;
        if (weight == 24) weight = 25;
        return weight;
    }

    public void deleteStaticProvider(String id) {
        URL oldProvider = findProviderUrl(id);
        if (oldProvider == null) {
            throw new IllegalStateException("Provider was changed!");
        }
        registry.unregister(oldProvider);
    }

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

    public Provider findProvider(String id) {
        return SyncUtils.url2Provider(findProviderUrlPair(id));
    }

    public Pair<String, URL> findProviderUrlPair(String id) {
        return SyncUtils.filterFromCategory(getRegistryCache(), Constants.PROVIDERS_CATEGORY, id);
    }

    public List<String> findServices() {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (providerUrls != null) ret.addAll(providerUrls.keySet());
        return ret;
    }

    public List<String> findAddresses() {
        List<String> ret = new ArrayList<String>();

        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (null == providerUrls) return ret;

        for (Map.Entry<String, Map<String, URL>> e1 : providerUrls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                String app = u.getAddress();
                if (app != null) ret.add(app);
            }
        }

        return ret;
    }

    public List<String> findAddressesByApplication(String application) {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        for (Map.Entry<String, Map<String, URL>> e1 : providerUrls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                if (application.equals(u.getParameter(Constants.APPLICATION_KEY))) {
                    String addr = u.getAddress();
                    if (addr != null) ret.add(addr);
                }
            }
        }

        return ret;
    }

    public List<String> findAddressesByService(String service) {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (null == providerUrls) return ret;

        for (Map.Entry<String, URL> e2 : providerUrls.get(service).entrySet()) {
            URL u = e2.getValue();
            String app = u.getAddress();
            if (app != null) ret.add(app);
        }

        return ret;
    }

    public List<String> findApplicationsByServiceName(String service) {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (null == providerUrls) return ret;

        Map<String, URL> value = providerUrls.get(service);
        if (value == null) {
            return ret;
        }
        for (Map.Entry<String, URL> e2 : value.entrySet()) {
            URL u = e2.getValue();
            String app = u.getParameter(Constants.APPLICATION_KEY);
            if (app != null) ret.add(app);
        }

        return ret;
    }

    public List<Provider> findByService(String serviceName) {
        return SyncUtils.url2ProviderList(findProviderUrlByService(serviceName));
    }

    public List<Provider> findByAppandService(String app, String serviceName) {
        return SyncUtils.url2ProviderList(findProviderUrlByAppandService(app, serviceName));
    }

    private Map<String, URL> findProviderUrlByService(String service) {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY);
        filter.put(SyncUtils.SERVICE_FILTER_KEY, service);

        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    public List<Provider> findAll() {
        return SyncUtils.url2ProviderList(findAllProviderUrl());
    }

    private Map<String, URL> findAllProviderUrl() {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY);
        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    public List<Provider> findByAddress(String providerAddress) {
        return SyncUtils.url2ProviderList(findProviderUrlByAddress(providerAddress));
    }

    public Map<String, URL> findProviderUrlByAddress(String address) {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY);
        filter.put(SyncUtils.ADDRESS_FILTER_KEY, address);

        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    public List<String> findServicesByAddress(String address) {
        List<String> ret = new ArrayList<String>();

        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (providerUrls == null || address == null || address.length() == 0) return ret;

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

    public List<String> findApplications() {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (providerUrls == null) return ret;

        for (Map.Entry<String, Map<String, URL>> e1 : providerUrls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                String app = u.getParameter(Constants.APPLICATION_KEY);
                if (app != null) ret.add(app);
            }
        }

        return ret;
    }

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
        filter.put(Constants.APPLICATION_KEY, app);
        filter.put(SyncUtils.SERVICE_FILTER_KEY, service);
        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }



    private Map<String, URL> findProviderUrlByApplication(String application) {
        Map<String, String> filter = new HashMap<>();
        filter.put(Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY);
        filter.put(Constants.APPLICATION_KEY, application);
        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    public List<String> findServicesByApplication(String application) {
        List<String> ret = new ArrayList<String>();

        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (providerUrls == null || application == null || application.length() == 0) return ret;

        for (Map.Entry<String, Map<String, URL>> e1 : providerUrls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                if (application.equals(u.getParameter(Constants.APPLICATION_KEY))) {
                    ret.add(e1.getKey());
                    break;
                }
            }
        }

        return ret;
    }

    public List<String> findMethodsByService(String service) {
        List<String> ret = new ArrayList<String>();

        ConcurrentMap<String, Map<String, URL>> providerUrls = getRegistryCache().get(Constants.PROVIDERS_CATEGORY);
        if (providerUrls == null || service == null || service.length() == 0) return ret;

        Map<String, URL> providers = providerUrls.get(service);
        if (null == providers || providers.isEmpty()) return ret;

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

}
