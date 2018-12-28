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
      <v-flex xs12>
        <search id="serviceSearch" v-model="filter" label="Search by service name" :submit="search"></search>
      </v-flex>
      <v-flex xs12>
        <h3>Methods</h3>
      </v-flex>
      <v-flex xs12>
        <v-data-table :headers="headers" :items="methods" hide-actions class="elevation-1">
          <template slot="items" slot-scope="props">
            <td>{{ props.item.name }}</td>
            <td><v-chip xs v-for="(type, index) in props.item.parameterTypes" :key="index" label>{{ type }}</v-chip></td>
            <td><v-chip label>{{ props.item.returnType }}</v-chip></td>
            <td class="text-xs-right">
              <v-tooltip bottom>
                <v-icon small
                        class="mr-2"
                        color="blue"
                        slot="activator"
                        @click="toTest(props.item)">input</v-icon>
                <span>Try it</span>
              </v-tooltip>
            </td>
          </template>
        </v-data-table>
      </v-flex>
    </v-layout>

    <v-dialog v-model="modal.enable" width="1000px" persistent>
      <v-card>
        <v-card-title>
          <span class="headline">Test {{ modal.method }}</span>
        </v-card-title>
        <v-container grid-list-xl fluid>
          <v-layout row>
            <v-flex lg6>
              <json-editor v-model="modal.json" />
            </v-flex>
            <v-flex lg6>
              <json-editor v-model="modal.json" />
            </v-flex>
          </v-layout>
        </v-container>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="darken-1"
                 flat
                 @click="modal.enable = false">Close</v-btn>
          <v-btn color="primary"
                 depressed
                 @click="test">Execute</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script>
  import JsonEditor from '@/components/public/JsonEditor'
  import Search from '@/components/public/Search'

  export default {
    name: 'ServiceTest',
    data () {
      return {
        filter: '',
        headers: [
          {
            text: 'Method Name',
            value: 'method',
            sortable: false
          },
          {
            text: 'Parameter List',
            value: 'parameter',
            sortable: false
          },
          {
            text: 'Return Type',
            value: 'returnType',
            sortable: false
          },
          {
            text: '',
            value: 'operation',
            sortable: false
          }
        ],
        service: null,
        methods: [],
        modal: {
          method: null,
          enable: false,
          types: null,
          json: []
        }
      }
    },
    methods: {
      search () {
        if (!this.filter) {
          this.filter = document.querySelector('#serviceSearch').value.trim()
          if (!this.filter) {
            return
          }
        }
        this.$router.push({
          path: 'test',
          query: { service: this.filter }
        })
        this.$axios.get('/service/' + this.filter).then(response => {
          this.service = response.data
          if (this.service.hasOwnProperty('metadata')) {
            this.methods = this.service.metadata.methods
          }
        }).catch(error => {
          this.showSnackbar('error', error.response.data.message)
        })
      },
      toTest (item) {
        Object.assign(this.modal, {
          enable: true,
          method: item.name
        })
        this.modal.json = []
        this.modal.types = item.parameterTypes
        item.parameterTypes.forEach((i, index) => {
          this.modal.json.push(this.getType(i))
        })
      },
      test () {
        this.$axios.post('/test', {
          service: this.service.metadata.canonicalName,
          method: this.modal.method,
          types: this.modal.types,
          params: this.modal.json
        }).then(response => {
          console.log(response)
        })
      },
      getType (type) {
        if (type.indexOf('java.util.List') === 0) {
          return []
        } else if (type.indexOf('java.util.Map') === 0) {
          return []
        } else {
          return ''
        }
      }
    },
    mounted: function () {
      let query = this.$route.query
      let filter = null
      Object.keys(query).forEach(function (key) {
        if (key === 'service') {
          filter = query[key]
        }
      })
      if (filter !== null) {
        this.filter = filter
        this.search()
      }
    },
    components: {
      JsonEditor,
      Search
    }
  }
</script>
