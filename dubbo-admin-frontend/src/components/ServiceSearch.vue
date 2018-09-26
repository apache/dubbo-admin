<!--
  - Licensed to the Apache Software Foundation (ASF) under one or more
  - contributor license agreements.  See the NOTICE file distributed with
  - this work for additional information regarding copyright ownership.
  - The ASF licenses this file to You under the Apache License, Version 2.0
  - (the "License"); you may not use this file except in compliance with
  - the License.  You may obtain a copy of the License at
  -
  -     http://www.apache.org/licenses/LICENSE-2.0
  -
  - Unless required by applicable law or agreed to in writing, software
  - distributed under the License is distributed on an "AS IS" BASIS,
  - WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  - See the License for the specific language governing permissions and
  - limitations under the License.
  -->

<template>
  <v-container id="search" grid-list-xl fluid >
    <v-layout row wrap>
      <v-flex lg12>
        <v-card flat color="transparent">
          <v-card-text>
            <v-form>
              <v-layout row wrap>
                  <v-text-field label="Search dubbo service"
                                :hint="hint"
                                :suffix="queryBy"
                                v-model="filter"></v-text-field>

                  <v-menu bottom left class="hidden-xs-only">
                    <v-btn
                      slot="activator"
                      icon>
                      <v-icon>unfold_more</v-icon>
                    </v-btn>

                    <v-list>
                      <v-list-tile
                        v-for="(item, i) in items"
                        :key="i"
                        @click="selected = i">
                        <v-list-tile-title>{{ item.title }}</v-list-tile-title>
                      </v-list-tile>
                    </v-list>
                  </v-menu>
                  <v-btn @click="submit" color="primary" large>Search</v-btn>
              </v-layout>
            </v-form>

          </v-card-text>
        </v-card>
      </v-flex>
    </v-layout>

    <v-flex lg12>
      <v-card>
        <v-toolbar card dense color="transparent">
          <v-toolbar-title><span class="headline">Search Result</span></v-toolbar-title>
          <v-spacer></v-spacer>
          <v-btn icon>
            <v-icon>more_vert</v-icon>
          </v-btn>
        </v-toolbar>

      <v-card-text class="pa-0">
        <template>
          <v-data-table
            hide-actions
            class="elevation-0 table-striped"
            :headers="headers"
            :items="services"
          >
            <template slot="items" slot-scope="props">
              <td>{{props.item.serviceName}}</td>
              <td>{{props.item.group}}</td>
              <td>{{props.item.appName}}</td>
              <td><v-btn small color='primary' :href='getHref(props.item.serviceName, props.item.appName)'>Detail</v-btn></td>
            </template>
          </v-data-table>
        </template>
        <v-divider></v-divider>
      </v-card-text>
      </v-card>
    </v-flex>
  </v-container>
</template>
<script>
  import {AXIOS} from './http-common'

  export default {
    data: () => ({
      items: [
        {id: 0, title: 'service name'},
        {id: 1, title: 'IP'},
        {id: 2, title: 'application'}
      ],
      selected: 0,
      services: [],
      filter: '',
      headers: [
        {
          text: 'Service Name',
          value: 'service',
          align: 'left'
        },
        {
          text: 'Group',
          value: 'group',
          align: 'left'
        },
        {
          text: 'Application',
          value: 'application',
          align: 'left'
        },
        {
          text: 'Operation',
          value: 'operation',
          sortable: false
        }
      ]
    }),
    computed: {
      queryBy () {
        return 'by ' + this.items[this.selected].title
      },
      hint () {
        if (this.selected === 0) {
          return 'Full qualified class name with service version, e.g. org.apache.dubbo.HelloService:1.0.0'
        } else if (this.selected === 1) {
          return 'Find all services provided by the target server on the specified IP address'
        } else if (this.selected === 2) {
          return 'Input an application name to find all services provided by one particular application.'
        }
      }
    },
    methods: {
      getHref: function (service, app) {
        return '/#/serviceDetail?service=' + service + '&app=' + app
      },
      submit () {
        let pattern = this.items[this.selected].title
        this.search(this.filter, pattern, true)
      },
      search: function (filter, pattern, rewrite) {
        let service = {}
        service.filter = filter
        service.pattern = pattern
        AXIOS.post('service/search', service)
          .then(response => {
            this.services = response.data
            if (rewrite) {
              this.$router.push({path: 'service', query: {filter: filter, pattern: pattern}})
            }
          })
      }
    },
    mounted: function () {
      let query = this.$route.query
      let filter = ''
      let pattern = ''
      Object.keys(query).forEach(function (key) {
        if (key === 'filter') {
          filter = query[key]
        }
        if (key === 'pattern') {
          pattern = query[key]
        }
      })
      if (filter !== '' && pattern !== '') {
        this.filter = filter
        if (pattern === 'service name') {
          this.selected = 0
        } else if (pattern === 'application') {
          this.selected = 2
        } else if (pattern === 'IP') {
          this.selected = 1
        }
        this.search(filter, pattern, false)
      }
    }

  }
</script>

