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

import io.github.bonigarcia.wdm.WebDriverManager;
import org.apache.commons.lang3.StringUtils;
import org.apache.dubbo.admin.pages.LoginPage;
import org.fluentlenium.adapter.junit.FluentTest;
import org.fluentlenium.core.annotation.Page;
import org.junit.BeforeClass;
import org.openqa.selenium.WebDriver;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class BaseIT extends FluentTest {
    private Logger logger = LoggerFactory.getLogger(this.getClass());

    protected static WebDriver driver;
    protected static String BASE_URL;

    @Page
    LoginPage loginPage;

    public BaseIT() {
        setWebDriver("chrome");
        setScreenshotPath("target/screens/");
        setScreenshotMode(TriggerMode.AUTOMATIC_ON_FAIL);
    }

    @BeforeClass
    public static void beforeClass() {
        WebDriverManager.chromedriver().setup();

        BASE_URL = StringUtils.defaultString(System.getenv("BASEURL"), "http://localhost:8082");
    }

    @Override
    public String getBaseUrl() {
        return BASE_URL;
    }


    public void autoLogin() {
        goTo(loginPage);

        try {
            await().untilPredicate(fluentControl -> loginPage.url().contains("login"));

            loginPage.loginWithRoot();
        } catch (Exception ignore) {
            logger.info("already log in");
        }
    }
}
