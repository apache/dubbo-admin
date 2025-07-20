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

import type { RouterMeta } from '@/router/RouterMeta'
import type { RouteRecordRaw } from 'vue-router'
import LayoutTab from '../layout/tab/layout_tab.vue'
import _ from 'lodash'
import AppTabHeaderSlot from '@/views/resources/applications/slots/AppTabHeaderSlot.vue'
import ServiceTabHeaderSlot from '@/views/resources/services/slots/ServiceTabHeaderSlot.vue'
import InstanceTabHeaderSlot from '@/views/resources/instances/slots/InstanceTabHeaderSlot.vue'
import AddConditionRuleTabHeaderSlot from '@/views/traffic/routingRule/slots/addConditionRuleTabHeaderSlot.vue'
import ConditionRuleDetailTabHeaderSlot from '@/views/traffic/routingRule/slots/conditionRuleDetailTabHeaderSlot.vue'
import UpdateConditionRuleTabHeaderSlot from '@/views/traffic/routingRule/slots/updateConditionRuleTabHeaderSlot.vue'
import AddTagRuleTabHeaderSlot from '@/views/traffic/tagRule/slots/addTagRuleTabHeaderSlot.vue'
import TagRuleDetailTabHeaderSlot from '@/views/traffic/tagRule/slots/tagRuleDetailTabHeaderSlot.vue'
import UpdateTagRuleTabHeaderSlot from '@/views/traffic/tagRule/slots/updateTagRuleTabHeaderSlot.vue'
import AddDCTabHeaderSlot from '@/views/traffic/dynamicConfig/slots/addDCTabHeaderSlot.vue'
import UpdateDCTabHeaderSlot from '@/views/traffic/dynamicConfig/slots/updateDCTabHeaderSlot.vue'

export declare type RouteRecordType = RouteRecordRaw & {
  key?: string
  name: string
  children?: RouteRecordType[]
  meta?: RouterMeta
}

export const routes: Readonly<RouteRecordType[]> = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/Login.vue'),
    meta: {
      skip: true
    },
    children: []
  },
  {
    path: '/',
    name: 'Root',
    redirect: '/home',
    component: () => import('../layout/index.vue'),
    meta: {
      skip: true
    },
    children: [
      {
        path: '/home',
        name: 'homePage',
        component: () => import('../views/home/index.vue'),
        meta: {
          icon: 'carbon:web-services-cluster'
        }
      },
      {
        path: '/resources',
        name: 'resources',
        meta: {
          icon: 'carbon:web-services-cluster'
        },
        children: [
          {
            path: '/applications',
            name: 'applications',
            component: LayoutTab,
            redirect: 'list',
            meta: {
              tab_parent: true,
              slots: {
                header: AppTabHeaderSlot
              }
            },
            children: [
              {
                path: '/list',
                name: 'router.resource.app.list',
                component: () => import('../views/resources/applications/index.vue'),
                meta: {
                  hidden: true
                }
              },
              {
                path: '/detail/:pathId',
                name: 'applicationDomain.detail',
                component: () => import('../views/resources/applications/tabs/detail.vue'),
                meta: {
                  tab: true,
                  icon: 'tabler:list-details',
                  back: '/resources/applications/list'
                }
              },
              {
                path: '/instance/:pathId',
                name: 'applicationDomain.instance',
                component: () => import('../views/resources/applications/tabs/instance.vue'),
                meta: {
                  tab: true,
                  icon: 'ooui:instance-ltr',
                  back: '/resources/applications/list'
                }
              },
              {
                path: '/service/:pathId',
                name: 'applicationDomain.service',
                component: () => import('../views/resources/applications/tabs/service.vue'),
                meta: {
                  tab: true,
                  icon: 'carbon:web-services-definition',
                  back: '/resources/applications/list'
                }
              },
              {
                path: '/monitor/:pathId',
                name: 'applicationDomain.monitor',
                component: () => import('../views/resources/applications/tabs/monitor.vue'),
                meta: {
                  tab: true,
                  icon: 'material-symbols-light:monitor-heart-outline',
                  back: '/resources/applications/list'
                }
              },
              {
                path: '/tracing/:pathId',
                name: 'applicationDomain.tracing',
                component: () => import('../views/resources/applications/tabs/tracing.vue'),
                meta: {
                  tab: true,
                  icon: 'game-icons:digital-trace',
                  back: '/resources/applications/list'
                }
              },
              {
                path: '/config/:pathId',
                name: 'applicationDomain.config',
                component: () => import('../views/resources/applications/tabs/config.vue'),
                meta: {
                  tab: true,
                  icon: 'material-symbols:settings',
                  back: '/resources/applications/list'
                }
              },
              {
                path: '/event/:pathId',
                name: 'applicationDomain.event',

                component: () => import('../views/resources/applications/tabs/event.vue'),
                meta: {
                  tab: true,
                  hidden: true,
                  icon: 'material-symbols:date-range',
                  back: '/resources/applications/list'
                }
              }
            ]
          },
          {
            path: '/instances',
            name: 'instances',
            component: LayoutTab,
            redirect: 'list',
            meta: {
              tab_parent: true,
              slots: {
                header: InstanceTabHeaderSlot
              }
            },
            children: [
              {
                path: '/list',
                name: 'router.resource.ins.list',
                component: () => import('../views/resources/instances/index.vue'),
                meta: {
                  hidden: true
                }
              },
              {
                path: '/detail/:appName/:pathId?',
                name: 'instanceDomain.details',
                component: () => import('../views/resources/instances/tabs/detail.vue'),
                meta: {
                  tab: true,
                  icon: 'tabler:list-details',
                  back: '/resources/instances/list'
                }
              },
              {
                path: '/monitor/:pathId/:appName',
                name: 'instanceDomain.monitor',
                component: () => import('../views/resources/instances/tabs/monitor.vue'),
                meta: {
                  tab: true,
                  icon: 'ooui:instance-ltr',
                  back: '/resources/instances/list'
                }
              },
              {
                path: '/linktracking/:pathId/:appName',
                name: 'instanceDomain.linkTracking',
                component: () => import('../views/resources/instances/tabs/linkTracking.vue'),
                meta: {
                  tab: true,
                  icon: 'material-symbols-light:monitor-heart-outline',
                  back: '/resources/instances/list'
                }
              },
              {
                path: '/configuration/:pathId/:appName',
                name: 'instanceDomain.configuration',
                component: () => import('../views/resources/instances/tabs/configuration.vue'),
                meta: {
                  tab: true,
                  icon: 'material-symbols:settings',
                  back: '/resources/instances/list'
                }
              },
              {
                path: '/event/:pathId/:appName',
                name: 'instanceDomain.event',
                component: () => import('../views/resources/instances/tabs/event.vue'),
                meta: {
                  tab: true,
                  hidden: true,
                  icon: 'material-symbols:date-range',
                  back: '/resources/instances/list'
                }
              }
            ]
          },
          {
            path: '/services',
            name: 'services',
            redirect: 'list',
            component: LayoutTab,
            meta: {
              tab_parent: true,
              slots: {
                header: ServiceTabHeaderSlot
              }
            },
            children: [
              {
                path: '/list',
                name: 'router.resource.svc.list',
                component: () => import('../views/resources/services/search.vue'),
                meta: {
                  hidden: true
                }
              },
              {
                path: '/distribution/:pathId/:group?/:version?',
                name: 'distribution',
                component: () => import('../views/resources/services/tabs/distribution.vue'),
                meta: {
                  tab: true,
                  back: '/resources/services/list'
                }
              },

              // Temporarily hidden
              // {
              //   path: '/detail/:pathId',
              //   name: 'detail',
              //   component: () => import('../views/resources/services/tabs/detail.vue'),
              //   meta: {
              //     tab: true
              //   }
              // },
              // {
              //   path: '/debug/:pathId/:group?/:version?',
              //   name: 'debug',
              //   component: () => import('../views/resources/services/tabs/debug.vue'),
              //   meta: {
              //     tab: true,
              //     back: '/resources/services/list'
              //   }
              // },
              {
                path: '/monitor/:pathId/:group?/:version?',
                name: 'monitor',
                component: () => import('../views/resources/services/tabs/monitor.vue'),
                meta: {
                  tab: true,
                  back: '/resources/services/list'
                }
              },
              {
                path: '/tracing/:pathId/:group?/:version?',
                name: 'tracing',
                component: () => import('../views/resources/services/tabs/tracing.vue'),
                meta: {
                  tab: true,
                  back: '/resources/services/list'
                }
              },
              {
                path: '/sceneConfig/:pathId/:group?/:version?',
                name: 'sceneConfig',
                component: () => import('../views/resources/services/tabs/sceneConfig.vue'),
                meta: {
                  tab: true,
                  back: '/resources/services/list'
                }
              },
              {
                path: '/event/:pathId/:group?/:version?',
                name: 'event',
                component: () => import('../views/resources/services/tabs/event.vue'),
                meta: {
                  tab: true,
                  hidden: true,
                  back: '/resources/services/list'
                }
              }
            ]
          }
        ]
      },
      {
        path: '/traffic',
        name: 'trafficManagement',
        meta: {
          icon: 'eos-icons:cluster-management'
        },
        children: [
          {
            path: '/routingRule',
            name: 'routingRule',
            redirect: 'index',
            component: LayoutTab,
            meta: {
              tab_parent: true,
              slots: {
                header: ConditionRuleDetailTabHeaderSlot
              }
            },
            children: [
              {
                path: '/index',
                name: 'routingRuleIndex',
                component: () => import('../views/traffic/routingRule/index.vue'),
                meta: {
                  hidden: true
                }
              },
              {
                path: '/formview/:ruleName',
                name: 'routingRuleDomain.formView',
                component: () => import('../views/traffic/routingRule/tabs/formView.vue'),
                meta: {
                  tab: true,
                  icon: 'oui:apm-trace'
                }
              },
              {
                path: '/yamlview/:ruleName',
                name: 'routingRuleDomain.YAMLView',
                component: () => import('../views/traffic/routingRule/tabs/YAMLView.vue'),
                meta: {
                  tab: true,
                  icon: 'oui:app-console'
                }
              }
            ]
          },
          {
            path: '/addRoutingRule',
            name: 'addRoutingRule',
            component: LayoutTab,
            redirect: 'addByFormView',
            meta: {
              tab_parent: true,
              hidden: true,
              slots: {
                header: AddConditionRuleTabHeaderSlot
              }
            },
            children: [
              {
                path: '/addByFormView',
                name: 'addRoutingRuleDomain.formView',
                component: () => import('../views/traffic/routingRule/tabs/addByFormView.vue'),
                meta: {
                  tab: true,
                  icon: 'oui:apm-trace'
                }
              },
              {
                path: '/addByYamlView',
                name: 'addRoutingRuleDomain.YAMLView',
                component: () => import('../views/traffic/routingRule/tabs/addByYAMLView.vue'),
                meta: {
                  tab: true,
                  icon: 'oui:app-console'
                }
              }
            ]
          },
          {
            path: '/updateRoutingRule',
            name: 'updateRoutingRule',
            component: LayoutTab,
            meta: {
              tab_parent: true,
              hidden: true,
              slots: {
                header: UpdateConditionRuleTabHeaderSlot
              }
            },
            children: [
              {
                path: '/updateByFormView/:ruleName',
                name: 'updateRoutingRuleDomain.formView',
                component: () => import('../views/traffic/routingRule/tabs/updateByFormView.vue'),
                meta: {
                  tab: true
                }
              },
              {
                path: '/updateByYAMLView/:ruleName',
                name: 'updateRoutingRuleDomain.YAMLView',
                component: () => import('../views/traffic/routingRule/tabs/updateByYAMLView.vue'),
                meta: {
                  tab: true
                }
              }
            ]
          },
          {
            path: '/tagRule',
            name: 'tagRule',
            redirect: 'index',
            component: LayoutTab,
            meta: {
              tab_parent: true,
              slots: {
                header: TagRuleDetailTabHeaderSlot
              }
            },
            children: [
              {
                path: '/index',
                name: 'tagRuleIndex',
                component: () => import('../views/traffic/tagRule/index.vue'),
                meta: {
                  hidden: true
                }
              },
              {
                path: '/formview/:ruleName',
                name: 'tagRuleDomain.formView',
                component: () => import('../views/traffic/tagRule/tabs/formView.vue'),
                meta: {
                  tab: true,
                  icon: 'oui:apm-trace',
                  back: '/traffic/tagRule'
                }
              },
              {
                path: '/yamlview/:ruleName',
                name: 'tagRuleDomain.YAMLView',
                component: () => import('../views/traffic/tagRule/tabs/YAMLView.vue'),
                meta: {
                  tab: true,
                  icon: 'oui:app-console'
                }
              }
            ]
          },
          {
            path: '/addTagRule',
            name: 'addTagRule',
            component: LayoutTab,
            meta: {
              tab_parent: true,
              hidden: true,
              slots: {
                header: AddTagRuleTabHeaderSlot
              }
            },
            children: [
              {
                path: '/addByFormView',
                name: 'addTagRuleDomain.formView',
                component: () => import('../views/traffic/tagRule/tabs/addByFormView.vue'),
                meta: {
                  tab: true,
                  icon: 'oui:apm-trace'
                }
              },
              {
                path: '/addByYAMLView',
                name: 'addTagRuleDomain.YAMLView',
                component: () => import('../views/traffic/tagRule/tabs/addByYAMLView.vue'),
                meta: {
                  tab: true,
                  icon: 'oui:app-console'
                }
              }
            ]
          },
          {
            path: '/updateTagRule',
            name: 'updateTagRule',
            component: LayoutTab,
            meta: {
              tab_parent: true,
              hidden: true,
              slots: {
                header: UpdateTagRuleTabHeaderSlot
              }
            },
            children: [
              {
                path: '/updateByFormView/:ruleName',
                name: 'updateTagRuleDomain.formView',
                component: () => import('../views/traffic/tagRule/tabs/updateByFormView.vue'),
                meta: {
                  // hidden: true,
                  tab: true
                }
              },
              {
                path: '/updateByYAMLView/:ruleName',
                name: 'updateTagRuleDomain.YAMLView',
                component: () => import('../views/traffic/tagRule/tabs/updateByYAMLView.vue'),
                meta: {
                  // hidden: true,
                  tab: true
                }
              }
            ]
          },
          {
            path: '/dynamicConfig',
            name: 'dynamicConfig',
            redirect: 'index',
            component: LayoutTab,
            meta: {
              tab_parent: true
            },
            children: [
              {
                path: '/index',
                name: 'dynamicConfigIndex',
                component: () => import('../views/traffic/dynamicConfig/index.vue'),
                meta: {
                  hidden: true
                }
              },
              {
                path: '/formview/:pathId/:isEdit',
                name: 'dynamicConfigDomain.formView',
                component: () => import('../views/traffic/dynamicConfig/tabs/formView.vue'),
                meta: {
                  tab: true,
                  icon: 'oui:apm-trace',
                  back: '/traffic/dynamicConfig',
                  slots: {
                    header: UpdateDCTabHeaderSlot
                  }
                }
              },
              {
                path: '/yamlview/:pathId/:isEdit',
                name: 'dynamicConfigDomain.YAMLView',
                component: () => import('../views/traffic/dynamicConfig/tabs/YAMLView.vue'),
                meta: {
                  tab: true,
                  icon: 'oui:app-console',
                  back: '/traffic/dynamicConfig',
                  slots: {
                    header: UpdateDCTabHeaderSlot
                  }
                }
              }
            ]
          }
          // {
          //   path: '/meshRule',
          //   name: 'meshRule',
          //   children: [
          //     {
          //       path: '/virtualService',
          //       name: 'virtualService',
          //       redirect: 'index',
          //       component: LayoutTab,
          //       meta: {
          //         tab_parent: true
          //       },
          //       children: [
          //         {
          //           path: '/index',
          //           name: 'virtualServiceIndex',
          //           component: () => import('../views/traffic/virtualService/index.vue'),
          //           meta: {
          //             hidden: true
          //           }
          //         },
          //         {
          //           path: '/formview/:ruleName',
          //           name: 'virtualServiceDomain.formView',
          //           component: () => import('../views/traffic/virtualService/tabs/formView.vue'),
          //           meta: {
          //             tab: true,
          //             icon: 'oui:apm-trace'
          //           }
          //         },
          //         {
          //           path: '/yamlview/:ruleName',
          //           name: 'virtualServiceDomain.YAMLView',
          //           component: () => import('../views/traffic/virtualService/tabs/YAMLView.vue'),
          //           meta: {
          //             tab: true,
          //             icon: 'oui:app-console'
          //           }
          //         }
          //       ]
          //     },
          //     {
          //       path: '/destinationRule',
          //       name: 'destinationRule',
          //       redirect: 'index',
          //       component: LayoutTab,
          //       meta: {
          //         tab_parent: true
          //       },
          //       children: [
          //         {
          //           path: '/index',
          //           name: 'destinationRuleIndex',
          //           component: () => import('../views/traffic/destinationRule/index.vue'),
          //           meta: {
          //             hidden: true
          //           }
          //         },
          //         {
          //           path: '/formview/:ruleName',
          //           name: 'destinationRuleDomain.formView',
          //           component: () => import('../views/traffic/destinationRule/tabs/formView.vue'),
          //           meta: {
          //             tab: true,
          //             icon: 'oui:apm-trace'
          //           }
          //         },
          //         {
          //           path: '/yamlview/:ruleName',
          //           name: 'destinationRuleDomain.YAMLView',
          //           component: () => import('../views/traffic/destinationRule/tabs/YAMLView.vue'),
          //           meta: {
          //             tab: true,
          //             icon: 'oui:app-console'
          //           }
          //         }
          //       ]
          //     }
          //   ]
          // }
        ]
      },
      {
        path: '/common',
        name: 'commonDemo',
        redirect: 'tab',
        meta: {
          hidden: true,
          icon: 'tdesign:play-demo'
        },
        children: [
          {
            path: '/tab',
            name: 'tabDemo',
            component: LayoutTab,
            redirect: 'index',
            meta: {
              tab_parent: true
            },
            children: [
              {
                path: '/index',
                name: 'tab_demo_index',
                component: () => import('../views/common/tab_demo/index.vue'),
                meta: {
                  hidden: true
                }
              },
              {
                path: '/tab1/:pathId',
                name: 'tab1',
                component: () => import('../views/common/tab_demo/tab1.vue'),
                meta: {
                  icon: 'simple-icons:podman',
                  tab: true
                }
              },
              {
                path: '/tab2/:pathId',
                name: 'tab2',
                component: () => import('../views/common/tab_demo/tab2.vue'),
                meta: {
                  icon: 'fontisto:docker',
                  tab: true
                }
              }
            ]
          },
          {
            path: '/placeholder',
            name: 'placeholder_demo',
            component: () => import('../views/common/placeholder_demo/index.vue'),
            meta: {}
          }
        ]
      }
    ]
  },
  {
    path: '/:catchAll(.*)',
    name: 'notFound',
    component: () => import('../views/error/notFound.vue'),
    meta: {
      skip: true
    }
  }
]

function handlePath(...paths: any[]) {
  return paths.join('/').replace(/\/+/g, '/')
}

function handleRoutes(
  routes: readonly RouteRecordType[] | undefined,
  parent: RouteRecordType | undefined
) {
  if (!routes) return
  for (const route of routes) {
    if (parent) {
      route.path = handlePath(parent?.path, route.path)
    }
    if (route.redirect) {
      route.redirect = handlePath(route.path, route.redirect || '')
    }

    if (route.meta) {
      route.meta._router_key = _.uniqueId('__router_key')
      route.meta.parent = parent
      // fixme, its really useful for tab_router judging how to  show tab
      route.meta.skip = route.meta.skip === true ? true : false
    } else {
      route.meta = {
        _router_key: _.uniqueId('__router_key'),
        skip: false
      }
    }
    handleRoutes(route.children, route)
  }
}

handleRoutes(routes, undefined)
