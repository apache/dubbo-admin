/*
 *
 *   Licensed to the Apache Software Foundation (ASF) under one or more
 *   contributor license agreements.  See the NOTICE file distributed with
 *   this work for additional information regarding copyright ownership.
 *   The ASF licenses this file to You under the Apache License, Version 2.0
 *   (the "License"); you may not use this file except in compliance with
 *   the License.  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *   Unless required by applicable law or agreed to in writing, software
 *   distributed under the License is distributed on an "AS IS" BASIS,
 *   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *   See the License for the specific language governing permissions and
 *   limitations under the License.
 *
 */
package org.apache.dubbo.admin;


import com.sun.jersey.spi.container.servlet.ServletContainer;
import org.apache.dubbo.admin.impl.consumer.RestApiService;
import org.apache.dubbo.config.spring.context.annotation.EnableDubbo;
import org.eclipse.jetty.server.Server;
import org.eclipse.jetty.servlet.ServletContextHandler;
import org.eclipse.jetty.servlet.ServletHolder;
import org.springframework.context.annotation.AnnotationConfigApplicationContext;
import org.springframework.context.annotation.ComponentScan;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.PropertySource;

public class ConsumerBootstrap {
    public static void main(String[] args) throws Exception {
        int port = 8282;
        try {
            port = Integer.parseInt(System.getenv("rest.api.port"));
        } catch (Exception ex) {
            //ignore
        }

        AnnotationConfigApplicationContext context = new AnnotationConfigApplicationContext(ConsumerBootstrap.ConsumerConfiguration.class);
        context.start();


        RestApiService.applicationContext = context;

        Server server = new Server(port);
        ServletHolder servlet = new ServletHolder(ServletContainer.class);

        servlet.setInitParameter("com.sun.jersey.config.property.packages", "org.apache.dubbo.admin.impl.consumer");
        servlet.setInitParameter("com.sun.jersey.api.json.POJOMappingFeature", "true");

        ServletContextHandler handler = new ServletContextHandler(ServletContextHandler.SESSIONS);
        handler.setContextPath("/");
        handler.addServlet(servlet, "/*");
        server.setHandler(handler);
        server.start();

        System.out.println("dubbo service init finish");
    }


    @Configuration
    @EnableDubbo(scanBasePackages = "org.apache.dubbo.admin.impl.consumer")
    @ComponentScan(basePackages = "org.apache.dubbo.admin.impl.consumer")
    @PropertySource("classpath:/spring/dubbo-consumer.properties")
    public static class ConsumerConfiguration {
    }

}
