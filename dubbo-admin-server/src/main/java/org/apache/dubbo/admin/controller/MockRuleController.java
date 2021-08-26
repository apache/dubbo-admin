package org.apache.dubbo.admin.controller;

import org.apache.dubbo.admin.annotation.Authority;
import org.apache.dubbo.admin.model.dto.GlobalMockRuleDTO;
import org.apache.dubbo.admin.model.dto.MockRuleDTO;
import org.apache.dubbo.admin.service.MockRuleService;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

/**
 * @author chenglu
 * @date 2021-08-24 15:48
 */
@Authority(needLogin = true)
@RestController
@RequestMapping("/api/{env}/mock/rule")
public class MockRuleController {

    @Autowired
    private MockRuleService mockRuleService;

    @PostMapping
    public boolean createOrUpdateMockRule(@RequestBody MockRuleDTO mockRule) {
        mockRuleService.createOrUpdateMockRule(mockRule);
        return true;
    }

    @DeleteMapping
    public boolean deleteMockRule(@RequestBody MockRuleDTO mockRule) {
        mockRuleService.deleteMockRuleById(mockRule.getId());
        return true;
    }

    @GetMapping("/list")
    public Page<MockRuleDTO> listMockRules(@RequestParam(required = false) String filter, Pageable pageable) {
        return mockRuleService.listMockRulesByPage(filter, pageable);
    }

    @GetMapping("/global")
    public GlobalMockRuleDTO getGlobalMockRule() {
        return mockRuleService.getGlobalMockRule();
    }

    @PostMapping("/global")
    public boolean changeGlobalMockRule(@RequestBody GlobalMockRuleDTO globalMockRule) {
        mockRuleService.changeGlobalMockRule(globalMockRule);
        return true;
    }
}
