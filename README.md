# Dubbo Admin

![GitHub Workflow Status](https://img.shields.io/github/workflow/status/apache/dubbo-admin/CI)
[![codecov](https://codecov.io/gh/apache/dubbo-admin/branch/develop/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-admin/branches/develop)
![license](https://img.shields.io/github/license/apache/dubbo-admin.svg)
[![Average time to resolve an issue](http://isitmaintained.com/badge/resolution/apache/dubbo-admin.svg)](http://isitmaintained.com/project/apache/dubbo-admin "Average time to resolve an issue")
[![Percentage of issues still open](http://isitmaintained.com/badge/open/apache/dubbo-admin.svg)](http://isitmaintained.com/project/apache/dubbo-admin "Percentage of issues still open")

[中文说明](README_ZH.md)
### Quick start

* prebuilt docker image https://hub.docker.com/r/apache/dubbo-admin
* quick start a live demo with [play-with-docker](https://labs.play-with-docker.com/?stack=https://raw.githubusercontent.com/apache/dubbo-admin/develop/docker/stack.yml#) (version:0.1.0)

### Screenshot

![index](https://raw.githubusercontent.com/apache/dubbo-admin/develop/doc/images/index.png)

### Service Governance  
service governance follows the version of Dubbo 2.7, and compatible for Dubbo 2.6, please refer to [here](https://github.com/apache/dubbo-admin/wiki/The-compatibility-of-service-governance)
### admin UI

- [Vue.js](https://vuejs.org) and [Vue Cli](https://cli.vuejs.org/)
- [dubbo-admin-ui/README.md](dubbo-admin-ui/README.md) for more detail
- Set npm **proxy mirror**: if you have network issue, you can set npm proxy mirror to speedup npm install: add `registry =https://registry.npm.taobao.org` to ~/.npmrc

### admin Server

* Standard spring boot project
* [configurations in application.properties](https://github.com/apache/dubbo-admin/wiki/Dubbo-Admin-configuration)


### Production Setup

1. Clone source code on develop branch `git clone https://github.com/apache/dubbo-admin.git`
2. Specify registry address in `dubbo-admin-server/src/main/resources/application.properties`
3. Build

    > - `mvn clean package -Dmaven.test.skip=true`  
4. Start 
    * `mvn --projects dubbo-admin-server spring-boot:run`  
    OR
    * `cd dubbo-admin-distribution/target`;   `java -jar dubbo-admin-0.1.jar`
5. Visit `http://localhost:8080`
6. Default username and password is `root`
---

### Development Setup
* Run admin server project
   backend is a standard spring boot project, you can run it in any java IDE
* Run admin ui project
  run with `npm run dev`.
* visit web page
  visit `http://localhost:8081`, frontend supports hot reload.
  
### Swagger support

Once deployed, you can check http://localhost:8080/swagger-ui.html to check all restful api and models


### License

Apache Dubbo admin is under the Apache 2.0 license, Version 2.0.
See [LICENSE](https://github.com/apache/dubbo-admin/blob/develop/LICENSE) for full license text.
