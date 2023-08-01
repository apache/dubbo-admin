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
import Accesslog from '@/components/resource/Accesslog'
import Arguments from '@/components/resource/Arguments'
import Gray from '@/components/resource/Gray'
// import Host from '@/components/resource/Host'
// import Isolation from '@/components/resource/Isolation'
import Mock from '@/components/resource/Mock'
import Region from '@/components/resource/Region'
import Retry from '@/components/resource/Retry'
import Timeout from '@/components/resource/Timeout'
import Weight from '@/components/resource/Weight'
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
          component: ServiceSearch
        },
        {
          path: '/serviceDetail',
          name: 'ServiceDetail',
          component: ServiceDetail
        },
        {
          path: '/testMethod',
          name: 'TestMethod',
          component: TestMethod
        },
        {
          path: '/resource/routingRule',
          name: 'RoutingRule',
          component: RoutingRule,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/resource/tagRule',
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
          path: '/resource/config',
          name: 'Overrides',
          component: Overrides
        },
        {
          path: '/test',
          name: 'ServiceTest',
          component: ServiceTest
        },
        {
          path: '/mock/dds',
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
          path: '/resource/accesslog',
          name: 'accesslog',
          component: Accesslog,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/resource/retry',
          name: 'retry',
          component: Retry,
          meta: {
            requireLogin: false
          }
        }, {
          path: '/resource/region',
          name: 'region',
          component: Region,
          meta: {
            requireLogin: false
          }
        },
        //  {
        //   path: '/resource/isolation',
        //   name: 'isolation',
        //   component: Isolation,
        //   meta: {
        //     requireLogin: false
        //   }
        // },
        {
          path: '/resource/weight',
          name: 'weight',
          component: Weight,
          meta: {
            requireLogin: false
          }
        }, {
          path: '/resource/arguments',
          name: 'arguments',
          component: Arguments,
          meta: {
            requireLogin: false
          }
        }, {
          path: '/resource/mock',
          name: 'mock',
          component: Mock,
          meta: {
            requireLogin: false
          }
        },
        // {
        //   path: '/resource/host',
        //   name: 'host',
        //   component: Host,
        //   meta: {
        //     requireLogin: false
        //   }
        // },
        {
          path: '/resource/timeout',
          name: 'timeout',
          component: Timeout,
          meta: {
            requireLogin: false
          }
        },
        {
          path: '/resource/gray',
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
