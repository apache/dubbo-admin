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

import Vue from 'vue'
import Router from 'vue-router'
import ServiceSearch from '@/components/ServiceSearch'
import ServiceDetail from '@/components/ServiceDetail'
import TestMethod from '@/components/test/TestMethod'
import RoutingRule from '@/components/governance/RoutingRule'
import TagRule from '@/components/governance/TagRule'
// import MeshRule from '@/components/governance/MeshRule'
// import AccessControl from '@/components/governance/AccessControl'
// import LoadBalance from '@/components/governance/LoadBalance'
// import WeightAdjust from '@/components/governance/WeightAdjust'
import Overrides from '@/components/governance/Overrides'
import ServiceTest from '@/components/test/ServiceTest'
import ApiDocs from '@/components/apiDocs/ApiDocs'
import ServiceMock from '@/components/test/ServiceMock'
import ServiceMetrics from '@/components/metrics/ServiceMetrics'
import ServiceRelation from '@/components/metrics/ServiceRelation'
import Management from '@/components/Management'
import Accesslog from '@/components/traffic/Accesslog'
import Arguments from '@/components/traffic/Arguments'
import Gray from '@/components/traffic/Gray'
// import Host from '@/components/traffic/Host'
// import Isolation from '@/components/traffic/Isolation'
import Mock from '@/components/traffic/Mock'
import Region from '@/components/traffic/Region'
import Retry from '@/components/traffic/Retry'
import Timeout from '@/components/traffic/Timeout'
import Weight from '@/components/traffic/Weight'
import Home from '@/components/Home'
import Index from '@/Index'
import Login from '@/Login'

const originalPush = Router.prototype.push
Router.prototype.push = function push (location) {
  return originalPush.call(this, location).catch(err => err)
}

Vue.use(Router)

export default new Router({
  routes: [
    {
      path: '/',
      name: 'Index',
      component: Index,
      children: [
        {
          path: '/service',
          name: 'ServiceSearch',
          component: ServiceSearch,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/home',
          name: 'Home',
          component: Home,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/serviceDetail',
          name: 'ServiceDetail',
          component: ServiceDetail,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/testMethod',
          name: 'TestMethod',
          component: TestMethod,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/traffic/routingRule',
          name: 'RoutingRule',
          component: RoutingRule,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/traffic/tagRule',
          name: 'TagRule',
          component: TagRule
        },
        // {
        //   path: '/governance/meshRule',
        //   name: 'MeshRule',
        //   component: MeshRule
        // },
        // {
        //   path: '/governance/access',
        //   name: 'AccessControl',
        //   component: AccessControl
        // },
        // {
        //   path: '/governance/loadbalance',
        //   name: 'LoadBalance',
        //   component: LoadBalance
        // },
        // {
        //   path: '/governance/weight',
        //   name: 'WeightAdjust',
        //   component: WeightAdjust
        // },
        {
          path: '/traffic/config',
          name: 'Overrides',
          component: Overrides
        },
        {
          path: '/test',
          name: 'ServiceTest',
          component: ServiceTest
        },
        {
          path: '/mock/rule',
          name: 'ServiceMock',
          component: ServiceMock,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/metrics/index',
          name: 'ServiceMetrics',
          component: ServiceMetrics,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/metrics/relation',
          name: 'ServiceRelation',
          component: ServiceRelation,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/management',
          name: 'Management',
          component: Management,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/apiDocs',
          name: 'apiDocs',
          component: ApiDocs,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/traffic/accesslog',
          name: 'accesslog',
          component: Accesslog,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/traffic/retry',
          name: 'retry',
          component: Retry,
          meta: {
            requireLogin: false
          }
        }, {
          path: '/traffic/region',
          name: 'region',
          component: Region,
          meta: {
            requireLogin: false
          }
        },
        //  {
        //   path: '/traffic/isolation',
        //   name: 'isolation',
        //   component: Isolation,
        //   meta: {
        //     requireLogin: false
        //   }
        // },
        {
          path: '/traffic/weight',
          name: 'weight',
          component: Weight,
          meta: {
            requireLogin: false
          }
        }, {
          path: '/traffic/arguments',
          name: 'arguments',
          component: Arguments,
          meta: {
            requireLogin: false
          }
        }, {
          path: '/traffic/mock',
          name: 'mock',
          component: Mock,
          meta: {
            requireLogin: false
          }
        },
        // {
        //   path: '/traffic/host',
        //   name: 'host',
        //   component: Host,
        //   meta: {
        //     requireLogin: false
        //   }
        // },
        {
          path: '/traffic/timeout',
          name: 'timeout',
          component: Timeout,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/traffic/gray',
          name: 'gray',
          component: Gray,
          meta: {
            requireLogin: false
          }
        }]
    }, {
      path: '/login',
      name: 'Login',
      component: Login,
      meta: {
        requireLogin: false
      }
    }

  ]
})
