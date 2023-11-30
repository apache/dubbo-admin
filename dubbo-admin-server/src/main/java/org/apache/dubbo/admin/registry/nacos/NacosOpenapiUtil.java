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
package org.apache.dubbo.admin.registry.nacos;

import com.alibaba.fastjson2.JSON;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.logger.Logger;
import org.apache.dubbo.common.logger.LoggerFactory;
import org.apache.dubbo.common.utils.StringUtils;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.MalformedURLException;
import java.util.Collections;
import java.util.List;

public class NacosOpenapiUtil {

    private static final Logger logger = LoggerFactory.getLogger(NacosOpenapiUtil.class);

    public static List<NacosData> getSubscribeAddressesWithHttpEndpoint(URL url, String serviceName) {
        // 定义Nacos OpenAPI的URL
        String nacosUrl = "http://" + url.getAddress() + "/nacos/v1/ns/service/subscribers?serviceName=" + serviceName;
        if (StringUtils.isNotEmpty(url.getParameter("namespace"))) {
            nacosUrl = nacosUrl + "&namespaceId=" + url.getParameter("namespace");
        }
        if (StringUtils.isNotEmpty(url.getParameter("group"))) {
            nacosUrl = nacosUrl + "&groupName=" + url.getParameter("group");
        }

        // 创建URL对象
        java.net.URL netUrl = null;
        try {
            netUrl = new java.net.URL(nacosUrl);
        } catch (MalformedURLException e) {
            throw new RuntimeException(e);
        }

        HttpURLConnection connection = null;
        try {

            // 创建HTTP连接
            connection = (HttpURLConnection) netUrl.openConnection();

            // 设置请求方法(GET或POST)
            connection.setRequestMethod("GET");

            // 发送请求并获取响应状态码
            int responseCode = connection.getResponseCode();

            if (responseCode == 200) {
                // 读取响应内容
                StringBuilder response = new StringBuilder();
                try (BufferedReader reader = new BufferedReader(new InputStreamReader(connection.getInputStream()))) {
                    String line;
                    while ((line = reader.readLine()) != null) {
                        response.append(line);
                    }
                } catch (Throwable t) {
                    logger.error("Error requesting nacos openapi, " + nacosUrl, t);
                }

                NacosResponse nacosResponse = JSON.parseObject(response.toString(), NacosResponse.class);
                return nacosResponse.getSubscribers();
            } else {
                logger.error("Error requesting nacos openapi, " + nacosUrl + ", error code is " + responseCode);
            }
        } catch (IOException e) {
            logger.error("Error requesting nacos openapi, " + nacosUrl, e);
        } finally {
            // 关闭连接
            if (connection != null) {
                connection.disconnect();
            }
        }
        return Collections.EMPTY_LIST;
    }
}
