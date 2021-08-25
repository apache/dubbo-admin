package org.apache.dubbo.admin.service;

import org.apache.dubbo.admin.model.dto.GlobalMockRuleDTO;
import org.apache.dubbo.admin.model.dto.MockRuleDTO;

import org.apache.dubbo.mock.api.MockResult;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;

/**
 * @author chenglu
 * @date 2021-08-24 15:49
 */
public interface MockRuleService {
    void createMockRule(MockRuleDTO mockRule);

    void deleteMockRuleById(Long id);

    void updateMockRule(MockRuleDTO mockRule);

    Page<MockRuleDTO> listMockRulesByPage(String filter, Pageable pageable);

    GlobalMockRuleDTO getGlobalMockRule();

    void changeGlobalMockRule(GlobalMockRuleDTO globalMockRule);

    MockResult getMockData(String interfaceName, String methodName, Object[] arguments);
}
