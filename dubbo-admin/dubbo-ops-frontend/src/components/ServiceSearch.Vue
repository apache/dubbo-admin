<template>
  <v-container id="search" grid-list-xl fluid >
    <v-layout row wrap>
      <v-layout row justify-center>
        <v-flex lg10>
          <v-text-field
            flat
            solo-inverted
            append-icon="search"
            @click:append="click"
          />
        </v-flex>
        <v-flex xs1
                class="pl-0 ml-0"
        >
          <v-select
            :items="dropdown_font"
            label="Select"
            v-model="service"
            solo-inverted
            single-line
          ></v-select>
        </v-flex>
      </v-layout>
      <v-flex sm12>
        <h3>Search Result</h3>
      </v-flex>
      <v-flex lg12>
        <v-data-table
          class="elevation-1"
          :headers="headers"
          :items="result"
        >
          <template slot="items" slot-scope="props">
            <td>{{props.item.service}}</td>
            <td>{{props.item.group}}</td>
            <td>{{props.item.application}}</td>
            <td>Detail</td>
          </template>
        </v-data-table>
      </v-flex>
    </v-layout>
  </v-container>
</template>
<script>
  export default {
    props: {
      result: {
        type: Array,
        default: () => [
          {
            service: 'com.alibaba.dubbo.com',
            group: 'dubbo',
            application: 'demo-provider'
          },
          {
            service: 'com.alibaba.sample',
            group: 'dubbo',
            application: 'demo-provider'
          },
          {
            service: 'com.taobao.core.engine',
            group: 'dubbo',
            application: 'demo-provider'
          }

        ]
      }
    },
    data: () => ({
      dropdown_font: [ 'Service', 'App', 'IP' ],
      service: 'Service',
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
    methods: {
      click: function () {
        console.log('aaa')
      }
    }
  }
</script>

