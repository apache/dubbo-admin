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

import org.apache.commons.io.FileUtils;
import org.openqa.selenium.OutputType;
import org.openqa.selenium.TakesScreenshot;
import org.openqa.selenium.WebDriver;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.IOException;

public class BaseIT {
    private Logger logger = LoggerFactory.getLogger(this.getClass());

    public void takeShot(WebDriver webDriver, String name) {
        TakesScreenshot scrShot = ((TakesScreenshot) webDriver);

        File SrcFile = scrShot.getScreenshotAs(OutputType.FILE);

        File DestFile = new File("target/screens/" + name + ".png");

        try {
            FileUtils.copyFile(SrcFile, DestFile);
        } catch (IOException e) {
            logger.info("#takeShot# take shot fail", e);
        }
    }
}
