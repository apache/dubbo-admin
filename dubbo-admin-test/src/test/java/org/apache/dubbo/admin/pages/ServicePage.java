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

import org.codehaus.plexus.util.StringUtils;
import org.fluentlenium.core.annotation.PageUrl;
import org.fluentlenium.core.domain.FluentWebElement;
import org.openqa.selenium.By;
import org.openqa.selenium.support.FindBy;

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
}
