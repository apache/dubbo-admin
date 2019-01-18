# Dubbo控制台

[![Build Status](https://travis-ci.org/apache/incubator-dubbo-ops.svg?branch=develop)](https://travis-ci.org/apache/incubator-dubbo-ops)
[![codecov](https://codecov.io/gh/apache/incubator-dubbo-ops/branch/develop/graph/badge.svg)](https://codecov.io/gh/apache/incubator-dubbo-ops)
![license](https://img.shields.io/github/license/apache/incubator-dubbo-ops.svg)

[English version](README.md).
### Demo地址
* http://47.91.207.147/#/service
* 该地址是`develop`分支的最新版本，在从源码构建之前，可以先尝试demo
### 页面截图

![index](https://raw.githubusercontent.com/apache/incubator-dubbo-ops/develop/doc/images/index.png)

### 服务治理  
服务治理的部分，按照Dubbo 2.7的格式进行配置，同时兼容Dubbo 2.6，详见[这里](https://github.com/apache/incubator-dubbo-ops/wiki/Dubbo-Admin%E6%9C%8D%E5%8A%A1%E6%B2%BB%E7%90%86%E5%85%BC%E5%AE%B9%E6%80%A7%E8%AF%B4%E6%98%8E)
### 前端部分

- 使用[Vue.js](https://vuejs.org)作为javascript框架，[Vuetify](https://vuetifyjs.com)作为UI框架
- [dubbo-admin-frontend/README.md](dubbo-admin-frontend/README.md)中有更详细的介绍

### 后端部分

* 标准spring boot工程
* **注意** 本分支依赖Dubbo2.7-SNAPSHOT版本，该Dubbo版本还未正式发布，因此如果发现依赖方面的错误，请清空本地库中的dubbo2.7相关文件
* [application.properties配置说明](https://github.com/apache/incubator-dubbo-ops/wiki/Dubbo-Admin%E9%85%8D%E7%BD%AE%E8%AF%B4%E6%98%8E)  
* **注意** 本分支依赖Dubbo2.7-SNAPSHOT版本，该Dubbo版本还未正式发布，因此如果发现依赖方面的错误，请清空本地库中的dubbo2.7相关文件
* 在项目根目录(incubator-dubbo-ops)第一次构建需要强制更新: `mvn -Dmaven.test.skip=true clean -U package`


### 生产环境配置

1. 下载代码: `git clone https://github.com/apache/incubator-dubbo-ops.git`
2. 在 `dubbo-admin-backend/src/main/resources/application-production.properties`中指定注册中心地址
3. 构建

    > - `mvn clean package`
4. 启动 
   * `mvn --projects dubbo-admin-backend spring-boot:run`   
   或者   
   * `cd dubbo-admin-backend/target; java -jar dubbo-admin-backend-0.0.1-SNAPSHOT.jar`
5. 访问 `http://localhost:8080`
---

### 开发环境配置
* 运行`dubbo admin backend`
   `dubbo admin backend`是一个标准的spring boot项目, 可以在任何java IDE中运行它
* 运行`dubbo admin frontend`
  `dubbo admin frontend`由npm管理和构建，在开发环境中，可以单独运行: `npm run dev`
* 页面访问
  访问 `http://localhost:8081`, 由于前后端分开部署，前端支持热加载，任何页面的修改都可以实时反馈，不需要重启应用。
 * 跨域问题
    为了方便开发，我们提供了这种前后端分离的部署模式，主要的好处是支持前端热部署，在这种模式下，前端会通过8080端口访问后端的restful api接口，获取数据, 这将导致跨域访问的问题。因此我们在`dubbo-admin-frontend/config/index.js`添加了支持跨域访问的配置,当前端通过`npm run dev`单独启动时，这些配置将被激活，允许跨域访问

### Swagger 支持

部署完成后，可以访问 http://localhost:8080/swagger-ui.html 来查看所有的restful api
