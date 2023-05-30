<!--
  ~ Licensed to the Apache Software Foundation (ASF) under one or more
  ~ contributor license agreements.  See the NOTICE file distributed with
  ~ this work for additional information regarding copyright ownership.
  ~ The ASF licenses this file to You under the Apache License, Version 2.0
  ~ (the "License"); you may not use this file except in compliance with
  ~ the License.  You may obtain a copy of the License at
  ~
  ~     http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing, software
  ~ distributed under the License is distributed on an "AS IS" BASIS,
  ~ WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  ~ See the License for the specific language governing permissions and
  ~ limitations under the License.
-->
<template>
    <v-container grid-list-xl fluid>
        <v-layout row wrap>
            <v-flex lg12>
        <Breadcrumb title="trafficTimeout" :items="breads"></breadcrumb>
      </v-flex>
      <v-flex lg12>
          <v-card flat color="transparent">
            <v-card-text>
              <v-form>
                <v-layout row wrap>
                  <v-combobox
                    id="serviceTestSearch"
                    :loading="searchLoading"
                    :items="typeAhead"
                    :search-input.sync="input"
                    v-model="filter"
                    flat
                    append-icon=""
                    hide-no-data
                    :hint="$t('testModule.searchServiceHint')"
                    :label="$t('placeholders.searchService')"
                    @keyup.enter="submit"
                  ></v-combobox>
                  <v-btn @click="submit" color="primary" large>{{ $t('search') }}</v-btn>
                  <v-btn @click="submit" color="primary" large>新建</v-btn>
                </v-layout>
              </v-form>
            </v-card-text>
          </v-card>
        </v-flex>
      <v-flex xs12>
        <v-card>
          <v-toolbar flat color="transparent" class="elevation-0">
            <v-toolbar-title><span class="headline">{{$t('trafficAccesslog')}}</span></v-toolbar-title>
            <v-spacer></v-spacer>
          </v-toolbar>
          <v-card-text class="pa-0">
            <v-data-table :headers="headers" :items="methods" hide-actions class="elevation-1">
              <template slot="items" slot-scope="props">
                <td>{{ props.item.name }}</td>
                <td>
                  <v-chip xs v-for="(type, index) in props.item.parameterTypes" :key="index" label>{{ type }}</v-chip>
                </td>
                <td>
                  <v-chip label>{{ props.item.returnType }}</v-chip>
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
          </v-card-text>
        </v-card>
      </v-flex>
        </v-layout>
    </v-container>
</template>
<script>
import Breadcrumb from '../public/Breadcrumb.vue'
export default {
  name: 'Accesslog',
  components: { Breadcrumb },
  data: () => ({
    breads: [
      {
        text: 'trafficManagement',
        href: ''
      },
      {
        text: 'trafficTimeout',
        href: ''
      }
    ]
  })
}
</script>
