package org.apache.dubbo.admin.authentication.impl;

import org.apache.dubbo.admin.annotation.Authority;
import org.apache.dubbo.admin.authentication.InterceptorAuthentication;
import org.apache.dubbo.admin.controller.UserController;
import org.apache.dubbo.admin.interceptor.AuthInterceptor;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.util.StringUtils;
import org.springframework.web.method.HandlerMethod;

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.lang.reflect.Method;


public class DefaultPreHandle implements InterceptorAuthentication {
    //make session timeout configurable
    //default to be an hour:1000 * 60 * 60
    @Value("${admin.check.sessionTimeoutMilli:3600000}")
    private long sessionTimeoutMilli;

    private AuthInterceptor authInterceptor = new AuthInterceptor();

    @Override
    public boolean authentication(HttpServletRequest request, HttpServletResponse response, Object handler) {
        HandlerMethod handlerMethod = (HandlerMethod) handler;
        Method method = handlerMethod.getMethod();
        Authority authority = method.getDeclaredAnnotation(Authority.class);
        if (null == authority) {
            authority = method.getDeclaringClass().getDeclaredAnnotation(Authority.class);
        }

        String authorization = request.getHeader("Authorization");
        if (null != authority && authority.needLogin()) {
            //check if 'authorization' is empty to prevent NullPointException
            //since UserController.tokenMap is an instance of ConcurrentHashMap.
            if (StringUtils.isEmpty(authorization)) {
                //While authentication is required and 'Authorization' string is missing in the request headers,
                //reject this request(http403).
                authInterceptor.rejectedResponse(response);
                return false;
            }

            UserController.User user = UserController.tokenMap.get(authorization);
            if (null != user && System.currentTimeMillis() - user.getLastUpdateTime() <= sessionTimeoutMilli) {
                user.setLastUpdateTime(System.currentTimeMillis());
                return true;
            }

            //while user not found, or session timeout, reject this request(http403).
            authInterceptor.rejectedResponse(response);
            return false;
        } else {
            return true;
        }
    }
}
