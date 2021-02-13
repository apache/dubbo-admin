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
package org.apache.dubbo.admin.pages;

import com.google.common.base.Preconditions;
import org.apache.commons.collections.CollectionUtils;
import org.codehaus.plexus.util.StringUtils;
import org.fluentlenium.core.annotation.PageUrl;
import org.fluentlenium.core.domain.FluentList;
import org.fluentlenium.core.domain.FluentWebElement;
import org.openqa.selenium.By;
import org.openqa.selenium.support.FindBy;

import java.util.concurrent.TimeUnit;

@PageUrl("/#/service")
public class ServicePage extends BasePage {
    @FindBy(css = "input#serviceSearch")
    private FluentWebElement serviceSearchInput;

    @FindBy(css = "button.primary")
    private FluentWebElement serviceSearchButton;

    @FindBy(css = "table.v-datatable tbody tr")
    private FluentWebElement serviceList;

    @FindBy(css = "div.v-content__wrap")
    private FluentWebElement basicContainer;

    @FindBy(css = "table.v-datatable tbody tr")
    private FluentWebElement testMethodList;

    @FindBy(css = "button#execute")
    private FluentWebElement testExecButton;

    @FindBy(css = "div[contenteditable='true']")
    private FluentWebElement testExecInputs;

    @FindBy(css = "div.it-test-method-result-container")
    private FluentWebElement testResultContainer;

    public ServicePage checkDetailForService(String fullName) {
        await().until(serviceSearchInput).displayed();

        serviceSearchInput.fill().with(fullName);

        serviceSearchButton.click();

        await().untilPredicate(p -> serviceList.asList().size() > 0);

        for (FluentWebElement row : serviceList.asList()) {
            for (FluentWebElement td : row.find(By.cssSelector("td"))) {
                if (StringUtils.contains(fullName, td.text())) {
                    row.find(By.cssSelector("a.success")).first().click();
                    break;
                }
            }
        }

        await().untilPredicate(p -> basicContainer.text().contains("dubbo-admin-integration-provider"));

        return this;
    }

    public ServicePage checkTestDetailForService(String fullName) {
        await().until(serviceSearchInput).displayed();

        serviceSearchInput.fill().with(fullName);

        serviceSearchButton.click();

        await().untilPredicate(p -> serviceList.asList().size() > 0);

        for (FluentWebElement row : serviceList.asList()) {
            for (FluentWebElement td : row.find(By.cssSelector("td"))) {
                if (StringUtils.contains(fullName, td.text())) {
                    row.find(By.cssSelector("a.v-btn--depressed")).first().click();
                    break;
                }
            }
        }

        await().untilPredicate(p -> testMethodList.asList().size() > 0);

        return this;
    }

    public ServicePage openTestDialogForMethod(String methodName) {
        for (FluentWebElement method : testMethodList.asList()) {
            FluentList<FluentWebElement> tds = method.find(By.tagName("td"));
            if (CollectionUtils.isNotEmpty(tds) && StringUtils.equalsIgnoreCase(tds.get(0).text(), methodName)) {
                method.find(By.cssSelector("span i")).click();
                break;
            }
        }

        await().until(testExecButton).clickable();

        return this;
    }

    public ServicePage executeTestMethodWithParam(String... params) {
        await().until(testExecInputs).displayed();

        Preconditions.checkArgument(params.length == testExecInputs.asList().size(), "params not match input list");

        for (int i = 0; i < testExecInputs.asList().size(); i++) {
            testExecInputs.asList().get(i).fill().withText(params[i]);
        }

        testResultContainer.click();

        try {
            //sleep for a few seconds to make input works
            TimeUnit.SECONDS.sleep(3);
        } catch (InterruptedException e) {
            //ignored
        }

        testExecButton.click();

        await().atMost(10,TimeUnit.SECONDS).untilPredicate(p-> !getTestMethodResult().contains("{0}"));

        return this;
    }

    public String getTestMethodResult() {
        for (FluentWebElement resultRow : testResultContainer.find(By.cssSelector("table.jsoneditor-values tr"))) {
            String resultAsString = resultRow.text().replace("\n", "");
            if (StringUtils.contains(resultAsString, "Result")) {
                return StringUtils.replaceOnce(resultAsString, "Result:", "");
            }
        }
        throw new RuntimeException("can't get test method result");
    }
}
