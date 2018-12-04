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

import org.apache.dubbo.admin.common.util.Pair;

public class OverrideConfig {
    private String side;
    private String[] addresses;
    private String[] providerAddresses;
    private Pair<String, Object>[] parameters;
    private String[] applications;
    private String[] services;

    public String getSide() {
        return side;
    }

    public void setSide(String side) {
        this.side = side;
    }

    public String[] getAddresses() {
        return addresses;
    }

    public void setAddresses(String[] addresses) {
        this.addresses = addresses;
    }

    public String[] getProviderAddresses() {
        return providerAddresses;
    }

    public void setProviderAddresses(String[] providerAddresses) {
        this.providerAddresses = providerAddresses;
    }

    public Pair<String, Object>[] getParameters() {
        return parameters;
    }

    public void setParameters(Pair<String, Object>[] parameters) {
        this.parameters = parameters;
    }

    public String[] getApplications() {
        return applications;
    }

    public void setApplications(String[] applications) {
        this.applications = applications;
    }

    public String[] getServices() {
        return services;
    }

    public void setServices(String[] services) {
        this.services = services;
    }
}
