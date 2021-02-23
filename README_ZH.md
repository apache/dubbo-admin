# Dubbo控制台

[![Build Status](https://travis-ci.org/apache/dubbo-admin.svg?branch=develop)](https://travis-ci.org/apache/dubbo-admin)
[![codecov](https://codecov.io/gh/apache/dubbo-admin/branch/develop/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-admin/branches/develop)
![license](https://img.shields.io/github/license/apache/dubbo-admin.svg)

[English version](README.md).
### 快速开始

* 预构建的Docker镜像 https://hub.docker.com/r/apache/dubbo-admin
* 快速启动一个演示环境 [play-with-docker](https://labs.play-with-docker.com/?stack=https://raw.githubusercontent.com/apache/dubbo-admin/develop/docker/stack.yml#) (版本:0.1.0)

### 页面截图

![index](https://raw.githubusercontent.com/apache/dubbo-admin/develop/doc/images/index.png)

### 服务治理  
服务治理的部分，按照Dubbo 2.7的格式进行配置，同时兼容Dubbo 2.6，详见[这里](https://github.com/apache/dubbo-admin/wiki/%E6%9C%8D%E5%8A%A1%E6%B2%BB%E7%90%86%E5%85%BC%E5%AE%B9%E6%80%A7%E8%AF%B4%E6%98%8E)
### 前端部分

- 使用[Vue.js](https://vuejs.org)作为javascript框架
- [dubbo-admin-ui/README.md](dubbo-admin-ui/README.md)中有更详细的介绍
- 设置 npm **代理镜像** : 如果遇到了网络问题，可以设置npm代理镜像来加速npm install的过程：在~/.npmrc中增加 `registry =https://registry.npm.taobao.org`

### 后端部分

* 标准spring boot工程
* [application.properties配置说明](https://github.com/apache/dubbo-admin/wiki/Dubbo-Admin%E9%85%8D%E7%BD%AE%E8%AF%B4%E6%98%8E)  


### 生产环境配置

1. 下载代码: `git clone https://github.com/apache/dubbo-admin.git`
2. 在 `dubbo-admin-server/src/main/resources/application.properties`中指定注册中心地址
3. 构建

    > - `mvn clean package -Dmaven.test.skip=true`  
4. 启动 
   * `mvn --projects dubbo-admin-server spring-boot:run`   
   或者   
   * `cd dubbo-admin-distribution/target; java -jar dubbo-admin-0.1.jar`
5. 访问 `http://localhost:8080`
---

### 开发环境配置
* 运行`dubbo admin server`
   `dubbo admin server`是一个标准的spring boot项目, 可以在任何java IDE中运行它
* 运行`dubbo admin ui`
  `dubbo admin ui`由npm管理和构建，在开发环境中，可以单独运行: `npm run dev`
* 页面访问
  访问 `http://localhost:8081`, 由于前后端分开部署，前端支持热加载，任何页面的修改都可以实时反馈，不需要重启应用。

### Swagger 支持

部署完成后，可以访问 http://localhost:8080/swagger-ui.html 来查看所有的restful api
