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
package com.alibaba.dubboadmin.registry.common.domain;

import java.util.List;

import com.alibaba.dubbo.common.Constants;
import com.alibaba.dubbo.common.URL;

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
    public static final String KEY_CONSUMER_HOST = "consumer.host";
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
        this.rule = rule;
        String[] rules = rule.split(" => ");
        if (rules.length != 2) {
            throw new IllegalArgumentException("Illegal Route Condition Rule");
        }
        this.matchRule = rules[0];
        this.filterRule = rules[1];
    }

    public String getMatchRule() {
        return matchRule;
    }

    public void setMatchRule(String matchRule) {
        this.matchRule = matchRule;
    }

    public String getFilterRule() {
        return filterRule;
    }

    public void setFilterRule(String filterRule) {
        this.filterRule = filterRule;
    }

    @java.lang.Override
    public String toString() {
        return "Route [parentId=" + parentId + ", name=" + name
                + ", serviceName=" + service + ", matchRule=" + matchRule
                + ", filterRule=" + filterRule + ", priority=" + priority
                + ", username=" + username + ", enabled=" + enabled + "]";
    }

    public URL toUrl() {
        String group = null;
        String version = null;
        String path = service;
        int i = path.indexOf("/");
        if (i > 0) {
            group = path.substring(0, i);
            path = path.substring(i + 1);
        }
        i = path.lastIndexOf(":");
        if (i > 0) {
            version = path.substring(i + 1);
            path = path.substring(0, i);
        }
        return URL.valueOf(Constants.ROUTE_PROTOCOL + "://" + Constants.ANYHOST_VALUE + "/" + path
                + "?" + Constants.CATEGORY_KEY + "=" + Constants.ROUTERS_CATEGORY
                + "&router=condition&runtime=false&enabled=" + isEnabled() + "&priority=" + getPriority() + "&force=" + isForce() + "&dynamic=false"
                + "&name=" + getName() + "&" + Constants.RULE_KEY + "=" + URL.encode(getMatchRule() + " => " + getFilterRule())
                + (group == null ? "" : "&" + Constants.GROUP_KEY + "=" + group)
                + (version == null ? "" : "&" + Constants.VERSION_KEY + "=" + version));
    }

}
