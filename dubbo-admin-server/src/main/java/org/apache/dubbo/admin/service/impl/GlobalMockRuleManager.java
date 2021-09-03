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

import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.registry.config.GovernanceConfiguration;

import com.google.gson.Gson;
import org.apache.dubbo.common.utils.StringUtils;
import org.apache.dubbo.mock.api.GlobalMockRule;
import org.apache.dubbo.mock.api.MockConstants;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import javax.annotation.PostConstruct;
import java.util.Set;

/**
 * @author chenglu
 * @date 2021-09-03 17:09
 */
@Component
public class GlobalMockRuleManager {

    private static final String CONFIG_KEY = Constants.CONFIG_KEY + Constants.PATH_SEPARATOR +
            MockConstants.ADMIN_MOCK_RULE_KEY;

    @Autowired
    private GovernanceConfiguration configuration;

    private GlobalMockRule globalMockRule = new GlobalMockRule();

    private Gson gson;

    @PostConstruct
    public void init() {
        gson = new Gson();
        refreshRule();
    }

    private void refreshRule() {
        String rule = configuration.getConfig(CONFIG_KEY);
        globalMockRule.getEnabledMockRules().clear();
        if (StringUtils.isBlank(rule)) {
            globalMockRule.setEnableMock(false);
            return;
        }
        GlobalMockRule newGlobalMockRule = gson.fromJson(rule, GlobalMockRule.class);
        globalMockRule.setEnableMock(newGlobalMockRule.getEnableMock());
        globalMockRule.getEnabledMockRules().addAll(newGlobalMockRule.getEnabledMockRules());
    }

    private void updateGlobalMockRule() {
        configuration.setConfig(CONFIG_KEY, new Gson().toJson(globalMockRule));
    }

    public void updateEnableMock(boolean enable) {
        boolean oldValue = globalMockRule.getEnableMock();
        if (oldValue == enable) {
            return;
        }
        globalMockRule.setEnableMock(enable);
        updateGlobalMockRule();
        refreshRule();
        if (oldValue == globalMockRule.getEnableMock()) {
            throw new IllegalStateException("Operation failed.");
        }
    }

    public void updateMockRule(String rule, boolean enable) {
        Set<String> rules = globalMockRule.getEnabledMockRules();
        if (enable) {
            if (rules.contains(rule)) {
                return;
            }
            rules.add(rule);
        } else {
            if (!rules.contains(rule)) {
                return;
            }
            rules.remove(rule);
        }
        updateGlobalMockRule();
    }

    public boolean getEnableMock() {
        return globalMockRule.getEnableMock();
    }

    public boolean isRuleEnable(String rule) {
        return globalMockRule.getEnabledMockRules().contains(rule);
    }
}
