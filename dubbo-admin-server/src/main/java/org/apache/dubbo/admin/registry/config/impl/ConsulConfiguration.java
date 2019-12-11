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

package org.apache.dubbo.admin.registry.config.impl;

import com.ecwid.consul.v1.ConsulClient;
import com.ecwid.consul.v1.Response;
import com.ecwid.consul.v1.kv.model.GetValue;
import org.apache.dubbo.admin.registry.config.GovernanceConfiguration;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.logger.Logger;
import org.apache.dubbo.common.logger.LoggerFactory;

public class ConsulConfiguration implements GovernanceConfiguration {
	private static final Logger logger = LoggerFactory.getLogger(ConsulConfiguration.class);
	private static final int DEFAULT_PORT = 8500;
	private static final String SLASH = "/";
	private URL url;
	private ConsulClient client;

	@Override
	public void init() {
		String host = this.url.getHost();
		int port = this.url.getPort() != 0 ? url.getPort() : DEFAULT_PORT;
		this.client = new ConsulClient(host, port);
	}

	@Override
	public void setUrl(URL url) {
		this.url = url;
	}

	@Override
	public URL getUrl() {
		return url;
	}

	@Override
	public String setConfig(String key, String value) {
		return setConfig(null, key, value);
	}

	@Override
	public String getConfig(String key) {
		return getConfig(null, key);
	}

	@Override
	public boolean deleteConfig(String key) {
		return deleteConfig(null, key);
	}

	@Override
	public String setConfig(String group, String key, String value) {
		if (group == null) {
			client.setKVValue(key, value);
			return value;
		}
		client.setKVValue(group + SLASH + key, value);
		return value;
	}

	@Override
	public String getConfig(String group, String key) {
		if (group == null) {
			Response<GetValue> response = client.getKVValue(key);
			if (response.getValue() == null) {
				return null;
			}
			return response.getValue().getDecodedValue();
		}
		Response<GetValue> response = client.getKVValue(group + SLASH + key);
		return response.getValue() == null ? null : response.getValue().getDecodedValue();
	}

	@Override
	public boolean deleteConfig(String group, String key) {
		try {
			if (group == null) {
				client.deleteKVValue(key);
				return true;
			}
			client.deleteKVValue(group + SLASH + key);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
			return false;
		}
		return true;
	}

	@Override
	public String getPath(String key) {
		return null;
	}

	@Override
	public String getPath(String group, String key) {
		return null;
	}

}
