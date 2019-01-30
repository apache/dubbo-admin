# Dubbo ops

[![Build Status](https://travis-ci.org/apache/incubator-dubbo-ops.svg?branch=develop)](https://travis-ci.org/apache/incubator-dubbo-ops)
[![codecov](https://codecov.io/gh/apache/incubator-dubbo-ops/branch/develop/graph/badge.svg)](https://codecov.io/gh/apache/incubator-dubbo-ops)
![license](https://img.shields.io/github/license/apache/incubator-dubbo-ops.svg)

[中文说明](README_ZH.md)
### Demo Address
* http://47.91.207.147/#/service
* this demo is the latest version of `develop` branch, you can try it before building from source code
### Screenshot

![index](https://raw.githubusercontent.com/apache/incubator-dubbo-ops/develop/doc/images/index.png)

### Service Governance  
service governance follows the version of Dubbo 2.7, and compatible for Dubbo 2.6, please refer to [here](https://github.com/apache/incubator-dubbo-ops/wiki/The-compatibility-of-service-governance)
### Frontend

- [Vue.js](https://vuejs.org) and [Vuetify](https://vuetifyjs.com)
- [dubbo-admin-frontend/README.md](dubbo-admin-frontend/README.md) for more detail

### Backend

* Standard spring boot project
* **NOTICE** this branch is based on Dubbo 2.7-SNAPSHOT, since this version is not published yet, so if you find a dependency error, please clear the dubbo2.7 related files in the local library
* [configurations in application.properties](https://github.com/apache/incubator-dubbo-ops/wiki/Dubbo-Admin-configuration)
* force update under root path(incubator-dubbo-ops) for the first building: `mvn -Dmaven.test.skip=true clean -U package`


### Production Setup

1. Clone source code on develop branch `git clone https://github.com/apache/incubator-dubbo-ops.git`
2. Specify registry address in `dubbo-admin-backend/src/main/resources/application-production.properties`
3. Build

    > - `mvn clean package`
4. Start 
    * `mvn --projects dubbo-admin-backend spring-boot:run`  
    OR
    * `cd dubbo-admin-backend/target`;   `java -jar dubbo-admin-backend-0.1.jar`
5. Visit `http://localhost:8080`
---

### Development Setup
* Run backend project
   backend is a standard spring boot project, you can run it in any java IDE
* Run frontend project
  run with `npm run dev`.
* visit web page
  visit `http://localhost:8081`, frontend supports hot reload.
 * CORS problem
    for the convenience of development, we deploy frontend and backend separately, so the frontend supports hot reload. In this mode, frontend will request `localhost:8080` to fetch data, this will cause a CORS problem, so we add a configuration in `dubbo-admin-frontend/config/index.js` to support CORS. This config will be activated under `npm run dev` mode.

### Swagger support

Once deployed, you can check http://localhost:8080/swagger-ui.html to check all restful api and models


### License

Apache Dubbo ops is under the Apache 2.0 license, Version 2.0.
See [LICENSE](https://github.com/apache/incubator-dubbo-ops/blob/develop/LICENSE) for full license text.
