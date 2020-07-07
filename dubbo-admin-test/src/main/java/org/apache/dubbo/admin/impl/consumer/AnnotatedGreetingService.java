package org.apache.dubbo.admin.impl.consumer;

import org.apache.dubbo.admin.api.GreetingService;
import org.apache.dubbo.config.annotation.DubboReference;
import org.springframework.stereotype.Service;

@Service
public class AnnotatedGreetingService {
    @DubboReference(version = "1.0.0")
    private GreetingService greetingService;

    public String sayHello(String name) {
        return greetingService.sayHello(name);
    }
}
