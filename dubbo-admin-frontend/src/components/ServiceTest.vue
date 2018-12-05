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
        <search v-model="filter" label="Search by service name" :submit="search"></search>
      </v-flex>
    </v-layout>

    <v-card>
      <v-card-text>
        <v-layout row>
          <v-flex xs6>
            <v-data-table :headers="headers" :items="methods" hide-actions>
              <template slot="items" slot-scope="props">
                <td>
                  <div>Name: {{ props.item.name }}</div>
                  <div>Return: {{ props.item.returnType }}</div>
                </td>
                <td></td>
              </template>
            </v-data-table>
          </v-flex>
          <v-flex xs6>
            <json-editor v-model="json" />
          </v-flex>
        </v-layout>
      </v-card-text>
    </v-card>
  </v-container>
</template>

<script>
  import JsonEditor from '@/components/public/JsonEditor'
  import Search from '@/components/public/Search'

  export default {
    name: 'ServiceTest',
    data () {
      return {
        filter: null,
        headers: [
          {
            text: 'Method',
            value: 'name',
            align: 'left'
          },
          {
            text: 'Operation',
            value: 'operation',
            sortable: false,
            width: '115px'
          }
        ],
        service: null,
        methods: [],
        json: {}
      }
    },
    methods: {
      search () {
        if (this.filter == null) {
          this.filter = ''
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
      }
    },
    components: {
      JsonEditor,
      Search
    }
  }
</script>
