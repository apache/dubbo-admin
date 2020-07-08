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
import org.junit.AfterClass;
import org.junit.BeforeClass;
import org.junit.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.WebDriver;
import org.openqa.selenium.firefox.FirefoxDriver;
import org.seleniumhq.selenium.fluent.FluentWebDriver;

import java.util.concurrent.TimeUnit;

public class LoginIT extends BaseIT {
    private static WebDriver driver;
    private static FluentWebDriver fwd;
    private static String BASE_URL;

    @BeforeClass
    public static void beforeClass() {
        WebDriverManager.firefoxdriver().setup();

        driver = new FirefoxDriver();

        driver.manage().timeouts().implicitlyWait(10, TimeUnit.SECONDS);
        fwd = new FluentWebDriver(driver);

        BASE_URL = StringUtils.defaultString(System.getenv("BASEURL"), "http://localhost:8082");
    }

    @AfterClass
    public static void afterClass() {
        driver.quit();
    }


    @Test
    public void shouldOpenLogin() {
        driver.get(BASE_URL + "/#/login");

        fwd.input(By.name("username")).sendKeys("root");
        fwd.input(By.cssSelector("input[type='password']")).sendKeys("root");

        this.takeShot(driver, "login");


        fwd.button(By.tagName("button")).click();
    }
}
