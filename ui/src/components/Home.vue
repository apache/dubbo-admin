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
        <Breadcrumb title="homePage" :items="breads"></breadcrumb>
          </v-flex>
        </v-layout>
        <v-container fluid grid-list-md>
    <v-data-iterator
      :items=clusterData
      content-tag="v-layout"
      hide-actions
      row
      wrap
    >
      <template v-slot:header>
        <v-toolbar
          class="mb-2"
          color="indigo darken-5"
          dark
          flat
        >
          <v-toolbar-title>ClusterOverview</v-toolbar-title>
        </v-toolbar>
      </template>
      <template v-slot:item="props">
        <v-flex
          xs12
          sm6
          md4
          lg4
        >
          <v-card>
            <v-card-title class="subheading font-weight-bold">{{ props.item.name }}</v-card-title>

            <v-divider></v-divider>

            <v-list dense>
              <v-list-tile>
                <v-list-tile-content>Number:</v-list-tile-content>
                <v-list-tile-content class="align-end">{{ props.item.number }}</v-list-tile-content>
              </v-list-tile>
            </v-list>
          </v-card>
        </v-flex>
      </template>
    </v-data-iterator>
        </v-container>
        <v-container fluid grid-list-md>
    <v-data-iterator
      :items=metaDate
      content-tag="v-layout"
      hide-actions
      row
      wrap
    >
      <template v-slot:header>
        <v-toolbar
          class="mb-2"
          color="indigo darken-5"
          dark
          flat
        >
          <v-toolbar-title>MetadataOverview</v-toolbar-title>
        </v-toolbar>
      </template>
      <template v-slot:item="props">
        <v-flex
          xs12
          sm6
          md4
          lg6
        >
          <v-card>
            <v-card-title class="subheading font-weight-bold">{{ props.item.name }}</v-card-title>

            <v-divider></v-divider>
            <template v-if="Array.isArray(props.item.value)">
              <v-list dense>
              <v-list-tile>
                <v-list-tile-content>Value:</v-list-tile-content>
                <v-list-tile-content class="align-end">{{joinArray(props.item.value) }}</v-list-tile-content>
              </v-list-tile>
              </v-list>
            </template>
            <template v-else>
              <v-list dense>
              <v-list-tile>
                <v-list-tile-content>Value:</v-list-tile-content>
                <v-list-tile-content class="align-end">{{ props.item.value }}</v-list-tile-content>
              </v-list-tile>
              </v-list>
            </template>
          </v-card>
        </v-flex>
      </template>
    </v-data-iterator>
        </v-container>
        <v-container fluid grid-list-md>
    <v-data-iterator
      :items=version
      content-tag="v-layout"
      hide-actions
      row
      wrap
    >
      <template v-slot:header>
        <v-toolbar
          class="mb-2"
          color="indigo darken-5"
          dark
          flat
        >
          <v-toolbar-title>VersionOverview</v-toolbar-title>
        </v-toolbar>
      </template>
      <template v-slot:item="props">
        <v-flex
          xs12
          sm6
          md4
          lg4
        >
          <v-card>
            <v-card-title class="subheading font-weight-bold">{{ props.item.name }}</v-card-title>

            <v-divider></v-divider>

            <v-list dense>
              <v-list-tile>
                <v-list-tile-content>Value:</v-list-tile-content>
                <v-list-tile-content class="align-end">{{ props.item.value }}</v-list-tile-content>
              </v-list-tile>
            </v-list>
          </v-card>
        </v-flex>
      </template>
    </v-data-iterator>
        </v-container>
    </v-container>
</template>
<script>
import Breadcrumb from './public/Breadcrumb.vue'
export default {
  name: 'ClusterOverview',
  components: { Breadcrumb },
  data: () => ({
    breads: [
      {
        text: 'homePage',
        href: ''
      }
    ],
    clusterData:[],
    version:[],
    metaDate:[],
  }),
  methods:{
    getCluster () {
      this.$axios.get('/metrics/cluster').then(response => {
        console.log(response)
        this.clusterData =  Object.entries(response.data.data).map(([name, number]) => ({ name, number }));
        console.log(this.clusterData)
      })
    },
    getVersion () {
      this.$axios.get('/version').then(response => {
        console.log(response)
        this.version =  Object.entries(response.data.data).map(([name, value]) => ({ name, value }));
        console.log(this.version)
      })
    },
    getMeta () {
      this.$axios.get('/metrics/metadata').then(response => {
        console.log(response)
        this.metaDate =  Object.entries(response.data.data).map(([name, value]) => ({ name, value }));
        console.log(this.metaDate)
      })
    },
    joinArray(arr) {
      return arr.join(', ');
    }
  },
  mounted(){
     this.getCluster();
     this.getVersion();
     this.getMeta();
  }
}
</script>
