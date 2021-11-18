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

package org.apache.dubbo.admin.model.domain;

import org.apache.dubbo.admin.model.dto.MockRuleDTO;

/**
 * The entity for database query and insert.
 */
public class MockRule {

    private Long id;

    private String serviceName;

    private String methodName;

    private String rule;

    private Boolean enable;

    public String getServiceName() {
        return serviceName;
    }

    public void setServiceName(String serviceName) {
        this.serviceName = serviceName;
    }

    public String getMethodName() {
        return methodName;
    }

    public void setMethodName(String methodName) {
        this.methodName = methodName;
    }

    public String getRule() {
        return rule;
    }

    public void setRule(String rule) {
        this.rule = rule;
    }

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    public Boolean getEnable() {
        return enable;
    }

    public void setEnable(Boolean enable) {
        this.enable = enable;
    }

    @java.lang.Override
    public String toString() {
        return "MockRule{" +
                "id=" + id +
                ", serviceName='" + serviceName + '\'' +
                ", methodName='" + methodName + '\'' +
                ", rule='" + rule + '\'' +
                ", enable=" + enable +
                '}';
    }


    public static MockRule toMockRule(MockRuleDTO mockRule) {
        MockRule rule = new MockRule();
        rule.setServiceName(mockRule.getServiceName().trim());
        rule.setMethodName(mockRule.getMethodName().trim());
        rule.setId(mockRule.getId());
        rule.setRule(mockRule.getRule());
        rule.setEnable(mockRule.getEnable());
        return rule;
    }
}
