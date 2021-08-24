package org.apache.dubbo.admin.provider;

import org.apache.dubbo.admin.service.MockRuleService;

import org.apache.dubbo.config.annotation.DubboService;
import org.apache.dubbo.mock.api.MockResult;
import org.apache.dubbo.mock.api.MockService;
import org.springframework.beans.factory.annotation.Autowired;

/**
 * @author chenglu
 * @date 2021-08-24 14:37
 */
@DubboService
public class MockServiceProvider implements MockService {

    @Autowired
    private MockRuleService mockRuleService;

    @Override
    public MockResult mock(String interfaceName, String methodName, Object[] arguments) {
        return mockRuleService.getMockData(interfaceName, methodName, arguments);
    }
}
