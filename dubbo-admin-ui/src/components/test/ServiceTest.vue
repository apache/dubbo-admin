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
  <v-container grid-list-xl fluid>
    <v-layout row wrap>
      <v-flex lg12>
        <breadcrumb title="serviceTest" :items="breads"></breadcrumb>
      </v-flex>
      <v-flex xs12>
        <v-autocomplete
          flat
          hide-no-data
          v-model="service"
          :loading="loading"
          :search-input.sync="filter"
          :hint="$t('testModule.searchServiceHint')"
          :items="services"
          :label="$t('placeholders.searchService')"
          persistent-hint
          @keyup.enter="search"
          clearable
        ></v-autocomplete>
      </v-flex>
      <v-flex xs12>
        <h3>{{$t('methods')}}</h3>
      </v-flex>
      <v-flex xs12>
        <v-data-table :headers="headers" :items="methods" hide-actions class="elevation-1">
          <template slot="items" slot-scope="props">
            <td>{{ props.item.name }}</td>
            <td>
              <v-chip xs v-for="(type, index) in props.item.parameterTypes" :key="index" label>{{ type }}</v-chip>
            </td>
            <td>
              <v-chip v-if="props.item.returnType!=null" label>{{ props.item.returnType }}</v-chip>
            </td>
            <td class="text-xs-right">
              <v-tooltip bottom>
                <v-btn
                  fab dark small color="blue" slot="activator"
                  :href="getHref(props.item.application, props.item.service, props.item.signature)"
                >
                  <v-icon>edit</v-icon>
                </v-btn>
                <span>{{$t('test')}}</span>
              </v-tooltip>
            </td>
          </template>
        </v-data-table>
      </v-flex>
    </v-layout>
  </v-container>
</template>

<script>
  import JsonEditor from '@/components/public/JsonEditor'
  import Search from '@/components/public/Search'
  import Breadcrumb from '@/components/public/Breadcrumb'

  export default {
    name: 'ServiceTest',
    components: {
      JsonEditor,
      Search,
      Breadcrumb
    },
    data () {
      return {
        filter: this.$route.query['service'] || '',
        breads: [
          {
            text: 'serviceSearch',
            href: '/test'
          }
        ],
        headers: [],
        timerID: null,
        service: null,
        methods: [],
        services: [],
        searchKey: this.$route.query['service'] || '*',
        loading: false
      }
    },
    methods: {
      setHeaders: function () {
        this.headers = [
          {
            text: this.$t('methodName'),
            value: 'method',
            sortable: false
          },
          {
            text: this.$t('parameterList'),
            value: 'parameter',
            sortable: false
          },
          {
            text: this.$t('returnType'),
            value: 'returnType',
            sortable: false
          },
          {
            text: '',
            value: 'operation',
            sortable: false
          }
        ]
      },
      search () {
        if (!this.filter) {
          return
        }
        this.service = this.filter
        this.services = [this.filter]
        this.$router.replace({
          query: {service: this.filter}
        })
        this.$axios.get('/service/' + this.filter.replace('/', '*')).then(response => {
          this.methods = []
          if (response.data.metadata) {
            let methods = this.service.metadata.methods
            for (let i = 0; i < methods.length; i++) {
              let method = {}
              let sig = methods[i].name + '~'
              let parameters = methods[i].parameterTypes
              let length = parameters.length
              for (let j = 0; j < length; j++) {
                sig = sig + parameters[j]
                if (j !== length - 1) {
                  sig = sig + ';'
                }
              }
              method.signature = sig
              method.name = methods[i].name
              method.parameterTypes = methods[i].parameterTypes
              method.returnType = methods[i].returnType
              method.service = response.data.service
              method.application = response.data.application
              this.methods.push(method)
            }
          } else if (response.data.providers.length > 0) {
            let provider = response.data.providers[0]
            let application = provider.application
            let service = provider.service
            let parametersArray = provider.parameters.split('&')
            for (var i = 0; i < parametersArray.length; i++) {
              let [key, value] = parametersArray[i].split('=')
              if (key === 'methods') {
                let methods = decodeURI(value)
                let methodsArray = methods.split(',')
                for (var j = 0; j < methodsArray.length; j++) {
                  let method = {}
                  method.signature = methodsArray[j]
                  method.name = methodsArray[j]
                  method.parameterTypes = null
                  method.returnType = null
                  method.service = service
                  method.application = application
                  this.methods.push(method)
                }
              }
            }
          }
        }).catch(error => {
          this.showSnackbar('error', error.response.data.message)
        })
      },
      searchServices () {
        if (this.timerID) {
          clearTimeout(this.timerID)
        }
        let filter = this.filter || ''
        // Simulated ajax query
        this.timerID = setTimeout(() => {
          if (filter) {
            this.loading = true
            this.services = this.$store.getters.getServiceItems(filter)
            this.loading = false
          } else {
            this.services = []
          }
          this.timerID = null
        }, 200)
      },
      getHref (application, service, method) {
        return `/#/testMethod?application=${application}&service=${service}&method=${method}`
      }
    },
    computed: {
      area () {
        return this.$i18n.locale
      }
    },
    watch: {
      area () {
        this.setHeaders()
      },
      filter () {
        this.searchServices()
      },
      searchKey () {
        this.search()
      }
    },
    mounted: function () {
      this.$store.dispatch('loadServiceItems')
      this.setHeaders()
      this.search()
    }
  }
</script>
<style>
  .v-breadcrumbs {
    padding-left: 0;
  }
</style>
