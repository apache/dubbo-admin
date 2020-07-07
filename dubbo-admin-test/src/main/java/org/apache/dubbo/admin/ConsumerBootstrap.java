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
import org.eclipse.jetty.server.Server;
import org.eclipse.jetty.servlet.ServletContextHandler;
import org.eclipse.jetty.servlet.ServletHolder;

public class ConsumerBootstrap {
    public static void main(String[] args) throws Exception {
        int port = 8282;
        try {
            port = Integer.parseInt(System.getenv("rest.api.port"));
        } catch (Exception ex) {
            //ignore
        }

        Server server = new Server(port); // 监听8282端口
        ServletHolder servlet = new ServletHolder(ServletContainer.class);

        servlet.setInitParameter("com.sun.jersey.config.property.packages", "org.apache.dubbo.admin.impl.consumer");
        servlet.setInitParameter("com.sun.jersey.api.json.POJOMappingFeature", "true");

        ServletContextHandler handler = new ServletContextHandler(ServletContextHandler.SESSIONS);
        handler.setContextPath("/");
        handler.addServlet(servlet, "/*");
        server.setHandler(handler);
        server.start();
        System.out.println("start...in " + port);
    }
}
