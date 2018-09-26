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
package org.apache.dubbo.admin.registry.common.domain;

import com.alibaba.dubbo.common.Constants;
import com.alibaba.dubbo.common.URL;

import java.util.List;

/**
 * Route
 *
 */
public class Route extends Entity {

    public static final String ALL_METHOD = "*";
    public static final String KEY_METHOD = "method";

    // WHEN KEY
    public static final String KEY_CONSUMER_APPLICATION = "consumer.application";
    public static final String KEY_CONSUMER_GROUP = "consumer.cluster";
    public static final String KEY_CONSUMER_VERSION = "consumer.version";
    public static final String KEY_CONSUMER_HOST = "host";
    public static final String KEY_CONSUMER_METHODS = "consumer.methods";
    public static final String KEY_PROVIDER_APPLICATION = "provider.application";

    // THEN KEY
    public static final String KEY_PROVIDER_GROUP = "provider.cluster";
    public static final String KEY_PROVIDER_PROTOCOL = "provider.protocol";
    public static final String KEY_PROVIDER_VERSION = "provider.version";
    public static final String KEY_PROVIDER_HOST = "provider.host";
    public static final String KEY_PROVIDER_PORT = "provider.port";
    private static final long serialVersionUID = -7630589008164140656L;
    private long parentId; //default 0

    private String name;

    private String service;

    private String rule;

    private String matchRule;

    private String filterRule;

    private int priority;

    private String username;

    private boolean enabled;

    private boolean force;

    private String version;

    private String group;

    private boolean dynamic;

    private boolean runtime;

    private List<Route> children;

    public Route() {
    }

    public Route(Long id) {
        super(id);
    }

    public int getPriority() {
        return priority;
    }

    public void setPriority(int priority) {
        this.priority = priority;
    }

    public String getUsername() {
        return username;
    }

    public void setUsername(String username) {
        this.username = username;
    }

    public long getParentId() {
        return parentId;
    }

    public void setParentId(long parentId) {
        this.parentId = parentId;
    }

    public List<Route> getChildren() {
        return children;
    }

    public void setChildren(List<Route> subRules) {
        this.children = subRules;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public boolean isEnabled() {
        return enabled;
    }

    public void setEnabled(boolean enabled) {
        this.enabled = enabled;
    }

    public boolean isDynamic() {
        return dynamic;
    }

    public void setDynamic(boolean dynamic) {
        this.dynamic = dynamic;
    }

    public boolean isRuntime() {
        return runtime;
    }

    public void setRuntime(boolean runtime) {
        this.runtime = runtime;
    }

    public String getVersion() {
        return version;
    }

    public void setVersion(String version) {
        this.version = version;
    }

    public String getGroup() {
        return group;
    }

    public void setGroup(String group) {
        this.group = group;
    }

    public boolean isForce() {
        return force;
    }

    public void setForce(boolean force) {
        this.force = force;
    }

    public String getService() {
        return service;
    }

    public void setService(String service) {
        this.service = service;
    }

    public String getRule() {
        return rule;
    }

    public void setRule(String rule) {
        this.rule = rule.trim();
        String[] rules = rule.split("=>");
        if (rules.length != 2) {
            if (rule.endsWith("=>")) {
                this.matchRule = rules[0].trim();
                this.filterRule = "";
            } else {
                throw new IllegalArgumentException("Illegal Route Condition Rule");
            }
        } else {
            this.matchRule = rules[0].trim();
            this.filterRule = rules[1].trim();
        }
    }

    public String getMatchRule() {
        return matchRule;
    }

    public void setMatchRule(String matchRule) {
        if (matchRule != null) {
            this.matchRule = matchRule.trim();
        } else {
            this.matchRule = matchRule;
        }
    }

    public String getFilterRule() {
        return filterRule;
    }

    public void setFilterRule(String filterRule) {
        if (filterRule != null) {
            this.filterRule = filterRule.trim();
        } else {
            this.filterRule = filterRule;
        }
    }

    @java.lang.Override
    public String toString() {
        return "Route [parentId=" + parentId + ", name=" + name
                + ", serviceName=" + service + ", matchRule=" + matchRule
                + ", filterRule=" + filterRule + ", priority=" + priority
                + ", username=" + username + ", enabled=" + enabled + "]";
    }

    public URL toUrl() {
//        if (filterRule != null && filterRule.endsWith("null")) {
//            filterRule = null;
//        } else {
//            filterRule = filterRule.trim();
//        }
        return URL.valueOf(Constants.ROUTE_PROTOCOL + "://" + Constants.ANYHOST_VALUE + "/" + getService()
                + "?" + Constants.CATEGORY_KEY + "=" + Constants.ROUTERS_CATEGORY
                + "&router=condition&runtime=" + isRuntime() + "&enabled=" + isEnabled() + "&priority=" + getPriority() + "&force=" + isForce() + "&dynamic=" + isDynamic()
                + "&name=" + getName() + "&" + Constants.RULE_KEY + "=" + URL.encode(getMatchRule() + " => " + getFilterRule())
                + (getGroup() == null ? "" : "&" + Constants.GROUP_KEY + "=" + getGroup())
                + (getVersion() == null ? "" : "&" + Constants.VERSION_KEY + "=" + getVersion()));
    }

}
