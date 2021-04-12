/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { Button, message, Input, Drawer } from 'antd';
import React, { useState, useRef } from 'react';
import { useIntl, FormattedMessage } from 'umi';
import { PageContainer, FooterToolbar } from '@ant-design/pro-layout';
import type { ProColumns, ActionType } from '@ant-design/pro-table';
import ProTable from '@ant-design/pro-table';
import type { TableListItem } from './data.d';
import { queryService} from './service';
import {NodeExpandOutlined} from '@ant-design/icons'

const Service: React.FC = () => {

  const actionRef = useRef<ActionType>();
  /** 国际化配置 */
  const intl = useIntl();

  /** 查询列表 */
  const fetchServiceList = async (params:any)=>{

    let param = {};
    param.page = params.current - 1;
    param.size = params.pageSize;
    param.pattern = 'service';
    if(params.service){
      param.filter = params.service;
    }else{
      param.filter = '*';
    }

    let serviceResponseData = await queryService(param);
    return {data:serviceResponseData.content,success:true,total:serviceResponseData.totalElements};
  }


  const columns: ProColumns<TableListItem>[] = [
    {
      title: (
        <FormattedMessage
          id="pages.service.searchresult.column.servicename"
          defaultMessage="服务名"
        />
      ),
      dataIndex: 'service',
      tip:'服务名称是唯一的 key'
    },
    {
      title: <FormattedMessage id="pages.service.searchresult.column.groupname" defaultMessage="组名" />,
      dataIndex: 'group',
      valueType: 'textarea',
      search:false
    },
    {
      title: <FormattedMessage id="pages.service.searchresult.column.version" defaultMessage="版本" />,
      dataIndex: 'version',
      hideInForm: true,
      search:false
    },
    {
      title: <FormattedMessage id="pages.service.searchresult.column.appname" defaultMessage="应用名" />,
      dataIndex: 'appName',
      hideInForm: true,
      search:false
    },
    {
      title: <FormattedMessage id="pages.searchTable.titleOption" defaultMessage="操作" />,
      dataIndex: 'option',
      valueType: 'option',
      render: (_, record) => [
        <Button type="primary" icon={<NodeExpandOutlined />}>
          <FormattedMessage id="pages.common.button.detail" defaultMessage="详情" />
        </Button>
      ],
    },
  ];

  return (
    <PageContainer title={intl.formatMessage({
      id: 'pages.service.title',
      defaultMessage: '查询服务',
    })}>
      <ProTable<TableListItem>
        headerTitle={intl.formatMessage({
          id: 'pages.service.searchresult.table.title',
          defaultMessage: '查询服务',
        })}
        actionRef={actionRef}
        rowKey={(item,index)=>{
          return item.service?(item.service+index):index+'';
        }}
        search={{
          labelWidth: 120,
          collapseRender:false
        }}
        request={(params, sorter, filter) =>fetchServiceList({ ...params, sorter,filter})}
        columns={columns}
      />
    </PageContainer>
  );
};

export default Service;
