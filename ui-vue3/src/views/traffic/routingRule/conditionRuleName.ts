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

// The backend only accepts condition rule names with this suffix.
export const CONDITION_RULE_SUFFIX = '.condition-router'

// conditionRuleName returns `<key>.condition-router` for an application-scope
// rule and `<key>:<version>:<group>.condition-router` for a service-scope rule.
export const conditionRuleName = (scope: string, key: string, version = '', group = ''): string =>
  scope === 'service'
    ? `${key}:${version}:${group}${CONDITION_RULE_SUFFIX}`
    : `${key}${CONDITION_RULE_SUFFIX}`
