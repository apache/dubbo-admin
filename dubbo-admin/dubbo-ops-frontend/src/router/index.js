import Vue from 'vue'
import Router from 'vue-router'
import ServiceSearch from '@/components/ServiceSearch'
import ServiceDetail from '@/components/ServiceDetail'
import RoutingRule from '@/components/RoutingRule'

Vue.use(Router)

export default new Router({
  routes: [
    {
      path: '/',
      name: 'ServiceSearch',
      component: ServiceSearch
    },
    {
      path: '/serviceDetail',
      name: 'ServiceDetail',
      component: ServiceDetail
    },
    {
      path: '/routingRule',
      name: 'RoutingRule',
      component: RoutingRule
    }
  ]
})
