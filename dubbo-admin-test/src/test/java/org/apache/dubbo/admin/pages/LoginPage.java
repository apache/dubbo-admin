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
import org.openqa.selenium.support.FindBy;

@PageUrl("/")
public class LoginPage extends BasePage {

    @FindBy(css = "input[name='username']")
    private FluentWebElement usernameInput;

    @FindBy(css = "input[type='password']")
    private FluentWebElement passwordInput;

    @FindBy(css = "button.primary")
    private FluentWebElement loginButton;

    @FindBy(css = "div.v-avatar")
    private FluentWebElement avatarButton;

    public LoginPage loginWithRoot() {
        await().until(usernameInput).displayed();

        usernameInput.fill().with("root");
        passwordInput.fill().with("root");

        loginButton.scrollToCenter();
        loginButton.click();

        return this;
    }

    public LoginPage logout() {
        await().until(avatarButton).clickable();

        return this;
    }
}
