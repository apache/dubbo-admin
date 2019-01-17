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

import java.io.IOException;
import org.apache.curator.framework.CuratorFramework;
import org.apache.curator.framework.CuratorFrameworkFactory;
import org.apache.curator.retry.RetryOneTime;
import org.apache.curator.test.TestingServer;
import org.junit.AfterClass;
import org.junit.BeforeClass;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.web.client.TestRestTemplate;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.context.junit4.SpringRunner;
import org.springframework.web.client.RestTemplate;

/**
 * Created by kezhenxu94 at 2019/1/16 17:41.
 *
 * @author kezhenxu94 (kezhenxu94 at 163 dot com)
 */
@ActiveProfiles("test")
@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
public abstract class AbstractSpringIntegrationTest {
  protected RestTemplate restTemplate = (new TestRestTemplate()).getRestTemplate();

  protected static TestingServer zkServer;
  protected static CuratorFramework zkClient;

  @Value("${local.server.port}")
  protected int port;

  @BeforeClass
  public static void beforeClass() throws Exception {
    zkServer = new TestingServer(2182, true);
    zkClient = CuratorFrameworkFactory.newClient(zkServer.getConnectString(), new RetryOneTime(2000));
    zkClient.start();
  }

  @AfterClass
  public static void afterClass() throws IOException {
    zkClient.close();
    zkServer.stop();
  }

  protected String url(final String path) {
    return "http://localhost:" + port + path;
  }
}
