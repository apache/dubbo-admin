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
package org.apache.dubbo.admin.common.util;

import org.apache.dubbo.admin.model.domain.*;
import org.apache.dubbo.admin.model.domain.Override;
import org.apache.dubbo.admin.model.dto.BalancingDTO;
import org.apache.dubbo.admin.model.dto.DynamicConfigDTO;
import org.apache.dubbo.admin.model.dto.WeightDTO;
import org.apache.dubbo.admin.model.store.OverrideConfig;
import org.apache.dubbo.admin.model.store.OverrideDTO;
import org.apache.dubbo.common.utils.StringUtils;

import java.util.*;

/**
 * OverrideUtils.java
 *
 */
public class OverrideUtils {
    public static List<Weight> overridesToWeights(List<Override> overrides) {
        List<Weight> weights = new ArrayList<Weight>();
        if (overrides == null) {
            return weights;
        }
        for (Override o : overrides) {
            if (StringUtils.isEmpty(o.getParams())) {
                continue;
            } else {
                Map<String, String> params = StringUtils.parseQueryString(o.getParams());
                for (Map.Entry<String, String> entry : params.entrySet()) {
                    if (entry.getKey().equals("weight")) {
                        Weight weight = new Weight();
                        weight.setAddress(o.getAddress());
                        weight.setId(o.getId());
                        weight.setHash(o.getHash());
                        weight.setService(o.getService());
                        weight.setWeight(Integer.valueOf(entry.getValue()));
                        weights.add(weight);
                    }
                }
            }
        }
        return weights;
    }

    public static OverrideConfig weightDTOtoConfig(WeightDTO weightDTO) {
        OverrideConfig overrideConfig = new OverrideConfig();
        overrideConfig.setType(Constants.WEIGHT);
        overrideConfig.setEnabled(true);
        overrideConfig.setSide(Constants.PROVIDER_SIDE);
        overrideConfig.setAddresses(weightDTO.getAddresses());
        Map<String, Object> parameters = new HashMap<>();
        parameters.put(Constants.WEIGHT, weightDTO.getWeight());
        overrideConfig.setParameters(parameters);
        return overrideConfig;
    }

    public static DynamicConfigDTO createFromOverride(OverrideDTO overrideDTO) {
        DynamicConfigDTO dynamicConfigDTO = new DynamicConfigDTO();
        dynamicConfigDTO.setConfigVersion(overrideDTO.getConfigVersion());
        List<OverrideConfig> configs = new ArrayList<>();
        for (OverrideConfig overrideConfig : overrideDTO.getConfigs()) {
            if (overrideConfig.getType() == null) {
                configs.add(overrideConfig);
            }
        }
        if (configs.size() == 0) {
            return null;
        }
        dynamicConfigDTO.setConfigs(configs);
        if (overrideDTO.getScope().equals(Constants.APPLICATION)) {
            dynamicConfigDTO.setApplication(overrideDTO.getKey());
        } else {
            dynamicConfigDTO.setService(overrideDTO.getKey());
        }
        dynamicConfigDTO.setEnabled(overrideDTO.isEnabled());
        return dynamicConfigDTO;
    }
    public static OverrideDTO createFromDynamicConfig(DynamicConfigDTO dynamicConfigDTO) {
        OverrideDTO overrideDTO = new OverrideDTO();
        if (StringUtils.isNotEmpty(dynamicConfigDTO.getApplication())) {
            overrideDTO.setScope(Constants.APPLICATION);
            overrideDTO.setKey(dynamicConfigDTO.getApplication());
        } else {
            overrideDTO.setScope(Constants.SERVICE);
            overrideDTO.setKey(dynamicConfigDTO.getService());
        }
        overrideDTO.setConfigVersion(dynamicConfigDTO.getConfigVersion());
        overrideDTO.setConfigs(dynamicConfigDTO.getConfigs());
        return overrideDTO;
    }

    public static OverrideConfig balanceDTOtoConfig(BalancingDTO balancingDTO) {
        OverrideConfig overrideConfig = new OverrideConfig();
        overrideConfig.setType(Constants.BALANCING);
        overrideConfig.setEnabled(true);
        overrideConfig.setSide(Constants.CONSUMER_SIDE);
        Map<String, Object> parameters = new HashMap<>();
        if (balancingDTO.getMethodName().equals("*")) {
            parameters.put("loadbalance", balancingDTO.getStrategy());
        } else {
            parameters.put(balancingDTO.getMethodName() + ".loadbalance", balancingDTO.getStrategy());
        }
        overrideConfig.setParameters(parameters);
        return overrideConfig;
    }

    public static WeightDTO configtoWeightDTO(OverrideConfig config, String scope, String key) {
        WeightDTO weightDTO = new WeightDTO();
        if (scope.equals(Constants.APPLICATION)) {
            weightDTO.setApplication(key);
        } else {
            weightDTO.setService(key);
        }
        weightDTO.setWeight((int)config.getParameters().get(Constants.WEIGHT));
        weightDTO.setAddresses(config.getAddresses());
        return weightDTO;
    }

    public static BalancingDTO configtoBalancingDTO(OverrideConfig config, String scope, String key) {
        BalancingDTO balancingDTO = new BalancingDTO();
        if (scope.equals(Constants.APPLICATION)) {
            balancingDTO.setApplication(key);
        } else {
            balancingDTO.setService(key);
        }
        for (Map.Entry<String, Object> entry : config.getParameters().entrySet()) {
            String k = entry.getKey();
            String method;
            if (k.contains(".")) {
                method = k.split("\\.")[0];
            } else {
                method = "*";
            }
            balancingDTO.setMethodName(method);
            balancingDTO.setStrategy((String)entry.getValue());
        }
        return balancingDTO;
    }

    public static Weight overrideToWeight(Override override) {
        List<Weight> weights = OverrideUtils.overridesToWeights(Arrays.asList(override));
        if (weights != null && weights.size() > 0) {
            return overridesToWeights(Arrays.asList(override)).get(0);
        }
        return null;
    }

    public static Override weightToOverride(Weight weight) {
        Override override = new Override();
        override.setId(weight.getId());
        override.setHash(weight.getHash());
        override.setAddress(weight.getAddress());
        override.setEnabled(true);
        override.setParams("weight=" + weight.getWeight());
        override.setService(weight.getService());
        return override;
    }

    public static List<LoadBalance> overridesToLoadBalances(List<Override> overrides) {
        List<LoadBalance> loadBalances = new ArrayList<>();
        if (overrides == null) {
            return loadBalances;
        }
        for (Override o : overrides) {
            if (StringUtils.isEmpty(o.getParams())) {
                continue;
            } else {
                Map<String, String> params = StringUtils.parseQueryString(o.getParams());
                for (Map.Entry<String, String> entry : params.entrySet()) {
                    if (entry.getKey().endsWith("loadbalance")) {
                        LoadBalance loadBalance = new LoadBalance();
                        String method = null;
                        if (entry.getKey().endsWith(".loadbalance")) {
                            method = entry.getKey().split(".loadbalance")[0];
                        } else {
                            method = "*";
                        }

                        loadBalance.setMethod(method);
                        loadBalance.setId(o.getId());
                        loadBalance.setHash(o.getHash());
                        loadBalance.setService(o.getService());
                        loadBalance.setStrategy(entry.getValue());
                        loadBalances.add(loadBalance);

                    }
                }
            }
        }
        return loadBalances;
    }

    public static LoadBalance overrideToLoadBalance(Override override) {
        List<LoadBalance> loadBalances = OverrideUtils.overridesToLoadBalances(Arrays.asList(override));
        if (loadBalances != null && loadBalances.size() > 0) {
            return loadBalances.get(0);
        }
        return null;
    }

    public static Override loadBalanceToOverride(LoadBalance loadBalance) {
        Override override = new Override();
        override.setId(loadBalance.getId());
        override.setHash(loadBalance.getHash());
        override.setService(loadBalance.getService());
        override.setEnabled(true);
        String method = loadBalance.getMethod();
        String strategy = loadBalance.getStrategy();
        if (StringUtils.isEmpty(method) || method.equals("*")) {
            override.setParams("loadbalance=" + strategy);
        } else {
            override.setParams(method + ".loadbalance=" + strategy);
        }
        return override;
    }

}
