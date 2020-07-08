/*
 *
 *   Licensed to the Apache Software Foundation (ASF) under one or more
 *   contributor license agreements.  See the NOTICE file distributed with
 *   this work for additional information regarding copyright ownership.
 *   The ASF licenses this file to You under the Apache License, Version 2.0
 *   (the "License"); you may not use this file except in compliance with
 *   the License.  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *   Unless required by applicable law or agreed to in writing, software
 *   distributed under the License is distributed on an "AS IS" BASIS,
 *   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *   See the License for the specific language governing permissions and
 *   limitations under the License.
 *
 */

package org.apache.dubbo.admin;

import org.apache.curator.RetryPolicy;
import org.apache.curator.framework.CuratorFramework;
import org.apache.curator.framework.CuratorFrameworkFactory;
import org.apache.curator.retry.ExponentialBackoffRetry;
import org.apache.dubbo.config.spring.context.annotation.EnableDubbo;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.AnnotationConfigApplicationContext;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.PropertySource;

import javax.annotation.PostConstruct;
import java.util.concurrent.CountDownLatch;

public class ProviderBootstrap {
    public static void main(String[] args) throws Exception {
        AnnotationConfigApplicationContext context = new AnnotationConfigApplicationContext(
                ProviderConfiguration.class);
        context.start();

        System.out.println("dubbo service started");
        new CountDownLatch(1).await();
    }

    @Configuration
    @EnableDubbo(scanBasePackages = "org.apache.dubbo.admin.impl.provider")
    @PropertySource("classpath:/spring/dubbo-provider.properties")
    static class ProviderConfiguration {
        @Value("${dubbo.config-center.address}")
        private String configCenterAddress;

        @PostConstruct
        public void init() throws Exception {
            System.out.println("Using address:" + this.configCenterAddress);

            RetryPolicy retryPolicy = new ExponentialBackoffRetry(1000, 3);

            CuratorFramework client = CuratorFrameworkFactory
                    .newClient(this.configCenterAddress.replace("zookeeper://", ""), 5000, 3000, retryPolicy);

            client.start();

            System.out.println("Checking config");
            if (client.checkExists().forPath("/dubbo/config/dubbo/dubbo.properties") == null) {
                String configAsString = String
                        .format("dubbo.registry.address=%s%sdubbo.metadata-report.address=%s", this.configCenterAddress,
                                System.lineSeparator(), this.configCenterAddress);
                client.create().creatingParentContainersIfNeeded()
                        .forPath("/dubbo/config/dubbo/dubbo.properties", configAsString.getBytes());
                System.out.println("Creating config");
            } else {
                byte[] bytes = client.getData().forPath("/dubbo/config/dubbo/dubbo.properties");
                System.out.println("Reading config: " + new String(bytes));
            }
        }
    }
}
