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

package org.apache.dubbo.admin;

import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.stereotype.Component;

@Component
@ConfigurationProperties(prefix = "admin")
public class DubboAdminProperties {
  private ConfigCenter configCenter;
  private Registry registry;
  private Metadata metadata;
  private Apollo apollo;

  public ConfigCenter getConfigCenter() {
    return configCenter;
  }

  public void setConfigCenter(final ConfigCenter configCenter) {
    this.configCenter = configCenter;
  }

  public Registry getRegistry() {
    return registry;
  }

  public void setRegistry(final Registry registry) {
    this.registry = registry;
  }

  public Metadata getMetadata() {
    return metadata;
  }

  public void setMetadata(final Metadata metadata) {
    this.metadata = metadata;
  }

  public Apollo getApollo() {
    return apollo;
  }

  public void setApollo(final Apollo apollo) {
    this.apollo = apollo;
  }

  public static class ConfigCenter {
    private String address;
    private String username;
    private String password;

    public String getAddress() {
      return address;
    }

    public void setAddress(final String address) {
      this.address = address;
    }

    public String getUsername() {
      return username;
    }

    public void setUsername(final String username) {
      this.username = username;
    }

    public String getPassword() {
      return password;
    }

    public void setPassword(final String password) {
      this.password = password;
    }
  }

  public static class Registry {
    private String address;
    private String group;

    public String getAddress() {
      return address;
    }

    public void setAddress(final String address) {
      this.address = address;
    }

    public String getGroup() {
      return group;
    }

    public void setGroup(final String group) {
      this.group = group;
    }
  }

  public static class Metadata {
    private String address;

    public String getAddress() {
      return address;
    }

    public void setAddress(final String address) {
      this.address = address;
    }
  }

  public static class Apollo {
    private String token;
    private String appId;
    private String env;
    private String cluster;
    private String namespace;

    public String getToken() {
      return token;
    }

    public void setToken(final String token) {
      this.token = token;
    }

    public String getAppId() {
      return appId;
    }

    public void setAppId(final String appId) {
      this.appId = appId;
    }

    public String getEnv() {
      return env;
    }

    public void setEnv(final String env) {
      this.env = env;
    }

    public String getCluster() {
      return cluster;
    }

    public void setCluster(final String cluster) {
      this.cluster = cluster;
    }

    public String getNamespace() {
      return namespace;
    }

    public void setNamespace(final String namespace) {
      this.namespace = namespace;
    }
  }

}
