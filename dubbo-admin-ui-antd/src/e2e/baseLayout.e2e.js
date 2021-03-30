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
const { uniq } = require('lodash');
const RouterConfig = require('../../config/config').default.routes;

const BASE_URL = `http://localhost:${process.env.PORT || 8000}`;

function formatter(routes, parentPath = '') {
  const fixedParentPath = parentPath.replace(/\/{1,}/g, '/');
  let result = [];
  routes.forEach((item) => {
    if (item.path) {
      result.push(`${fixedParentPath}/${item.path}`.replace(/\/{1,}/g, '/'));
    }
    if (item.routes) {
      result = result.concat(
        formatter(item.routes, item.path ? `${fixedParentPath}/${item.path}` : parentPath),
      );
    }
  });
  return uniq(result.filter((item) => !!item));
}

beforeEach(async () => {
  await page.goto(`${BASE_URL}`);
  await page.evaluate(() => {
    localStorage.setItem('antd-pro-authority', '["admin"]');
  });
});

describe('Ant Design Pro E2E test', () => {
  const testPage = (path) => async () => {
    await page.goto(`${BASE_URL}${path}`);
    await page.waitForSelector('footer', {
      timeout: 2000,
    });
    const haveFooter = await page.evaluate(
      () => document.getElementsByTagName('footer').length > 0,
    );
    expect(haveFooter).toBeTruthy();
  };

  const routers = formatter(RouterConfig);
  routers.forEach((route) => {
    it(`test pages ${route}`, testPage(route));
  });

  it('topmenu should have footer', async () => {
    const params = '?navTheme=light&layout=topmenu';
    await page.goto(`${BASE_URL}${params}`);
    await page.waitForSelector('footer', {
      timeout: 2000,
    });
    const haveFooter = await page.evaluate(
      () => document.getElementsByTagName('footer').length > 0,
    );
    expect(haveFooter).toBeTruthy();
  });
});
