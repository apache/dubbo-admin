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
