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
        <breadcrumb title="metrics" :items="breads"></breadcrumb>
      </v-flex>
      <v-flex xs12 >
        <search id="serviceSearch" v-model="filter" :submit="submit" :label="$t('searchSingleMetrics')"></search>
      </v-flex>
      <v-flex lg4 sm6 xs12>
        <v-card>
          <v-card-text>
            <mini-chart
              title="Monthly Sales"
              sub-title="10%"
              icon="trending_up"
              :data="dataset.monthVisit"
              :chart-color="color.blue.base"
              type="line"
            >
            </mini-chart>
            <mini-chart
              title="Monthly Sales"
              sub-title="10%"
              icon="trending_up"
              :data="dataset.monthVisit"
              :chart-color="color.blue.base"
              type="line"
            >
            </mini-chart>
          </v-card-text>
        </v-card>
      </v-flex>
      <v-flex lg4 sm6 xs12>
        <v-card>
          <v-card-text>
            <mini-chart
              title="Monthly Sales"
              sub-title="10%"
              icon="trending_up"
              :data="dataset.monthVisit"
              :chart-color="color.blue.base"
              type="line"
            >
            </mini-chart>
            <mini-chart
              title="Monthly Sales"
              sub-title="10%"
              icon="trending_up"
              :data="dataset.monthVisit"
              :chart-color="color.blue.base"
              type="line"
            >
            </mini-chart>
          </v-card-text>
        </v-card>
      </v-flex>
      <v-flex lg4 sm6 xs12>
        <v-card>
          <v-card-text>
            <mini-chart
              title="Monthly Sales"
              sub-title="10%"
              icon="trending_up"
              :data="dataset.monthVisit"
              :chart-color="color.blue.base"
              type="line"
            >
            </mini-chart>
            <mini-chart
              title="Monthly Sales"
              sub-title="10%"
              icon="trending_up"
              :data="dataset.monthVisit"
              :chart-color="color.blue.base"
              type="line"
            >
            </mini-chart>
          </v-card-text>
        </v-card>
      </v-flex>
      <v-flex sm12>
        <h3>{{$t('methodMetrics')}}</h3>
      </v-flex>
      <v-flex lg12 >
        <v-tabs
          class="elevation-1">
          <v-tab>
            {{$t('providers')}}
          </v-tab>
          <v-tab>
            {{$t('consumers')}}
          </v-tab>
          <v-tab-item>
            <v-data-table
              class="elevation-1"
              :headers="headers"
              :items="providerDetails"
            >
              <template slot="items" slot-scope="props">
                <td>{{props.item.service}}</td>
                <td>{{props.item.method}}</td>
                <td>{{props.item.qps}}</td>
                <td>{{props.item.rt}}</td>
                <td>{{props.item.successRate}}</td>
              </template>
            </v-data-table>
          </v-tab-item>
          <v-tab-item >
            <v-data-table
              class="elevation-1"
              :headers="headers"
              :items="consumerDetails"
            >
              <template slot="items" slot-scope="props">
                <td>{{props.item.service}}</td>
                <td>{{props.item.method}}</td>
                <td>{{props.item.qps}}</td>
                <td>{{props.item.rt}}</td>
                <td>{{props.item.successRate}}</td>
              </template>
            </v-data-table>
          </v-tab-item>
        </v-tabs>
      </v-flex>
    </v-layout>
  </v-container>
</template>

<script>
  import EChart from '@/util/echart'
  import Material from 'vuetify/es5/util/colors'
  import MiniChart from '@/components/public/MiniChart'
  import Breadcrumb from '@/components/public/Breadcrumb'
  import Search from '@/components/public/Search'
  import {
    monthVisitData,
    campaignData,
    locationData,
    StackData,
    SinData
  } from '@/api/chart'
  export default {
    name: 'ServiceMetrics',
    components: {
      MiniChart,
      EChart,
      Breadcrumb,
      Search
    },
    data () {
      return {
        selectedTab: 'tab-1',
        filter: '',
        headers: [],
        providerDetails: [
          {
            service: 'a.b.c.d',
            method: 'aaaa~ICS',
            qps: '0.58',
            rt: '111',
            successRate: '100%'
          },
          {
            service: 'a.b.c.f',
            method: 'bbbb~ICS',
            qps: '0.87',
            rt: '120',
            successRate: '90%'
          }

        ],
        consumerDetails: [],
        option: null,
        dataset: {
          sinData: SinData,
          monthVisit: monthVisitData,
          campaign: campaignData,
          location: locationData,
          stackData: StackData
        },
        color: Material,
        breads: [
          {
            text: 'metrics',
            href: ''
          }
        ]

      }
    },
    methods: {
      submit: function () {
      },
      setHeaders: function () {
        this.headers = [
          {
            text: this.$t('service'),
            value: 'service'
          },
          {
            text: this.$t('method'),
            value: 'method'
          },
          {
            text: this.$t('qps'),
            value: 'qps'
          },
          {
            text: this.$t('rt'),
            value: 'rt'
          },
          {
            text: this.$t('successRate'),
            value: 'successRate'
          }
        ]
      }
    },
    mounted: function () {
      this.setHeaders()
    }
  }
</script>

<style scoped>

</style>
