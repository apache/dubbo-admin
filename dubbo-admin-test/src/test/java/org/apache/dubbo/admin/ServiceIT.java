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

import org.apache.dubbo.admin.pages.ServicePage;
import org.fluentlenium.core.annotation.Page;
import org.junit.Test;

import static org.hamcrest.CoreMatchers.containsString;
import static org.hamcrest.MatcherAssert.assertThat;

public class ServiceIT extends BaseIT {
    @Page
    private ServicePage servicePage;

    @Test
    public void shouldCheckServiceInfo() {
        autoLogin();

        goTo(servicePage);

        servicePage.takeScreenshot("service-page.png");

        servicePage.checkDetailForService("org.apache.dubbo.admin.api.GreetingService:1.0.0");

        servicePage.takeScreenshot("service-detail.png");
    }

    @Test
    public void shouldTestService() {
        autoLogin();

        goTo(servicePage).checkTestDetailForService("org.apache.dubbo.admin.api.GreetingService:1.0.0")
                .takeScreenshot("service-test-list.png");

        servicePage.openTestDialogForMethod("sayHello").executeTestMethodWithParam("world")
                .takeScreenshot("service-test-detail.png");

        assertThat(servicePage.getTestMethodResult(), containsString("hello, world"));
    }
}
