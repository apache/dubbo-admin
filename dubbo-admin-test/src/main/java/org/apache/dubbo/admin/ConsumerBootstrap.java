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
