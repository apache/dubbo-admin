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

import { loginHandlers } from './handlers/login'
import { serviceHandlers } from './handlers/service'
import { serviceDetailHandlers } from './handlers/serviceDetail'
import { serviceDistributionHandlers } from './handlers/serviceDistribution'
import { appHandlers } from './handlers/app'
import { instanceHandlers } from './handlers/instance'
import { clusterHandlers } from './handlers/cluster'
import { globalSearchHandlers } from './handlers/globalSearch'
import { versionHandlers } from './handlers/version'
import { dynamicConfigHandlers } from './handlers/dynamicConfig'
import { routingRuleHandlers } from './handlers/routingRule'
import { tagRuleHandlers } from './handlers/tagRule'
import { ruleVersionHandlers } from './handlers/ruleVersion'
import { destinationRuleHandlers, virtualServiceHandlers } from './handlers/istio'
import { promQLHandlers } from './handlers/promQL'
import { serverHandlers } from './handlers/server'

import type { HttpHandler } from 'msw'

export const handlers: HttpHandler[] = [
  ...loginHandlers,
  ...serviceHandlers,
  ...serviceDetailHandlers,
  ...serviceDistributionHandlers,
  ...appHandlers,
  ...instanceHandlers,
  ...clusterHandlers,
  ...globalSearchHandlers,
  ...versionHandlers,
  ...dynamicConfigHandlers,
  ...routingRuleHandlers,
  ...tagRuleHandlers,
  ...ruleVersionHandlers,
  ...destinationRuleHandlers,
  ...virtualServiceHandlers,
  ...promQLHandlers,
  ...serverHandlers
]
