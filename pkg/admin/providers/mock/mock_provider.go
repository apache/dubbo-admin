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

package mock

import (
	"context"

	"github.com/apache/dubbo-admin/pkg/config/admin"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	dubboconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	dubbogo "dubbo.apache.org/dubbo-go/v3/config"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"github.com/apache/dubbo-admin/pkg/admin/mapper"
	"github.com/apache/dubbo-admin/pkg/admin/providers/mock/api"
	"github.com/apache/dubbo-admin/pkg/admin/services"
)

var _ api.MockServiceServer = (*MockServiceServer)(nil)

type MockServiceServer struct {
	api.UnimplementedMockServiceServer
	mockRuleService services.MockRuleService
}

func (s *MockServiceServer) GetMockData(ctx context.Context, req *api.GetMockDataReq) (*api.GetMockDataResp, error) {
	rule, enable, err := s.mockRuleService.GetMockData(ctx, req.ServiceName, req.MethodName)
	if err != nil {
		return nil, err
	}
	return &api.GetMockDataResp{
		Rule:   rule,
		Enable: enable,
	}, nil
}

func RunMockServiceServer(admin admin.Admin, dubboConfig dubbogo.RootConfig) {
	var mockRuleService services.MockRuleService = &services.MockRuleServiceImpl{
		MockRuleMapper: &mapper.MockRuleMapperImpl{},
		Logger:         logger.Logger(),
	}
	dubbogo.SetProviderService(&MockServiceServer{
		mockRuleService: mockRuleService,
	})

	builder := dubbogo.NewRootConfigBuilder().
		AddRegistry("zkRegistryKey", dubbogo.NewRegistryConfigBuilder().SetAddress(admin.Registry.Address).SetRegistryType(dubboconstant.RegistryTypeAll).
			Build()).SetApplication(dubbogo.NewApplicationConfigBuilder().SetName("dubbo-admin").Build())

	for k, v := range dubboConfig.Protocols {
		builder.AddProtocol(k, dubbogo.NewProtocolConfigBuilder().SetName(v.Name).SetPort(v.Port).Build())
	}

	rootConfig := builder.Build()

	if err := dubbogo.Load(dubbogo.WithRootConfig(rootConfig)); err != nil {
		panic(err)
	}

	select {}
}
