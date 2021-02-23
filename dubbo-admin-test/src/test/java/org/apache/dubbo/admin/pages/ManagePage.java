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

import org.fluentlenium.core.annotation.PageUrl;
import org.fluentlenium.core.domain.FluentWebElement;
import org.openqa.selenium.By;
import org.openqa.selenium.support.FindBy;

@PageUrl("/#/management")
public class ManagePage extends BasePage {
    @FindBy(css = "table.v-datatable tbody tr")
    private FluentWebElement configList;

    @FindBy(css = "div.ace_content")
    private FluentWebElement configCard;

    public String getConfigDetail() {
        return configCard.text();
    }

    public ManagePage showConfigDetailFor(String name) {
        await().untilPredicate(p -> configList.asList().size() > 0);

        for (FluentWebElement row : configList.asList()) {
            for (FluentWebElement td : row.find(By.cssSelector("td"))) {
                if (td.text().equalsIgnoreCase(name)) {
                    for (FluentWebElement i : row.find(By.tagName("i"))) {
                        if (i.text().equalsIgnoreCase("visibility")) {
                            i.click();
                            await().until(configCard).displayed();
                            return this;
                        }
                    }

                }
            }
        }
        throw new RuntimeException("can't load detail");
    }
}
