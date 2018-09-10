<!--
  - Licensed to the Apache Software Foundation (ASF) under one or more
  - contributor license agreements.  See the NOTICE file distributed with
  - this work for additional information regarding copyright ownership.
  - The ASF licenses this file to You under the Apache License, Version 2.0
  - (the "License"); you may not use this file except in compliance with
  -  he License.  You may obtain a copy of the License at
  -
  -      http://www.apache.org/licenses/LICENSE-2.0
  -
  -  Unless required by applicable law or agreed to in writing, software
  -  distributed under the License is distributed on an "AS IS" BASIS,
  -  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  -  See the License for the specific language governing permissions and
  -  limitations under the License.
  -->

<template>
  <v-container id="search" grid-list-xl fluid >
    <v-layout row wrap>
      <v-flex xs12 >
        <v-card flat>
          <v-card-text>
            <v-form>
              <v-layout row wrap>
                <!--<v-flex xs10>-->
                  <v-text-field label="Search dubbo service"
                                v-bind:suffix="queryBy"
                                v-model="filter"></v-text-field>
                <!--</v-flex>-->

                <!--<v-flex xs2>-->
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
                  <v-btn @click="submit" color="primary"  >Search</v-btn>
                <!--</v-flex>-->
              </v-layout>
            </v-form>

          </v-card-text>
        </v-card>
      </v-flex>
    </v-layout>
    <!--<v-flex sm12>-->
      <!--<h3>Search Result</h3>-->
    <!--</v-flex>-->
    <v-flex lg12>
      <v-toolbar class="elevation-1" flat color="white">
        <v-toolbar-title>Search Result</v-toolbar-title>
      </v-toolbar>
      <v-data-table
        class="elevation-1"
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
    </v-flex>
  </v-container>
</template>
<script>
  import {AXIOS} from './http-common'

  export default {
    data: () => ({
      items: [
        {title: 'service name'},
        {title: 'IP'},
        {title: 'application'}
      ],
      selected: 0,
      services: [],
      filter: '',
      headers: [
        {
          text: 'Service',
          value: 'service',
          class: 'font-weight-black'
        },
        {
          text: 'Group',
          value: 'group',
          class: 'font-weight-black'
        },
        {
          text: 'Application',
          value: 'application',
          class: 'font-weight-black'
        },
        {
          text: 'Operation',
          value: 'operation',
          class: 'font-weight-black'
        }
      ]
    }),
    computed: {
      queryBy () {
        return 'by ' + this.items[this.selected].title
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
        AXIOS.get('service/search?' + 'filter=' + filter + '&pattern=' + pattern)
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
        this.search(filter, pattern, false)
      }
    }

  }
</script>

