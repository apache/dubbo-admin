# Dubbo ops

[![Build Status](https://travis-ci.org/apache/incubator-dubbo-ops.svg?branch=master)](https://travis-ci.org/apache/incubator-dubbo-ops)
[![codecov](https://codecov.io/gh/apache/incubator-dubbo-ops/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/incubator-dubbo-ops)
![license](https://img.shields.io/github/license/apache/incubator-dubbo-ops.svg)

[中文说明](README_ZH.md)
### Screenshot

![index](https://raw.githubusercontent.com/apache/incubator-dubbo-ops/develop/doc/images/index.png)

### Frontend

- [Vue.js](https://vuejs.org) and [Vuetify](https://vuetifyjs.com)
- [dubbo-admin-frontend/README.md](dubbo-admin-frontend/README.md) for more detail

### Backend

* Configuration files  

> - `application.properties`  
>   The generic configuration, shared by `application-develop.properties` and `application-production.properties`
> - `application-production.properties` (default)  
>   The configuration for production
> - `application-develop.properties`  
>   The configuration for develop
> 


### Production Setup

1. Clone source code on develop branch
2. Specify registry address in `dubbo-admin-backend/src/resources/application-production.properties`
3. Build   

    * select configuration files via command line  
    
    > - `mvn clean package -Pproduction` will active production configuration(`application-production.properties`)
    > - `mvn clean package -Ddevelop` will active develop configuration(`application-develop.properties`)
4. Start `mvn --projects dubbo-admin-backend spring-boot:run`
5. Visit `http://localhost:8080`
---

### Development Setup
* Configuration in IDE  

   * Select configuration files in Intellij Idea 

      1. Choose profile file during project importing   
         1. In the **Import from Maven** page where IntelliJ IDEA displays the profiles, select `develop` maven profile: 
      ![profile](https://raw.githubusercontent.com/apache/incubator-dubbo-ops/develop/doc/images/profile-idea.jpg)
         2.  Select **Next** and finish import

      2.  Choose profile file in the Maven Projects tool window to activate profiles.  
          1. Open the Maven Projects tool window.  
          2. Click the Profiles node to open a list of declared profiles.  
          3. Select the appropriate checkboxes to activate `develop` maven profile.
      
    * Select configuration files in Eclipse
        1. import project
        2. In **Project Explorer**, right click `dubbo-admin-backend`
        3. Choose **Maven**->**Select Maven Profiles**
        4. Select `develop` maven profile
        ![profile-eclipse](https://raw.githubusercontent.com/apache/incubator-dubbo-ops/develop/doc/images/profile-eclipse.jpg)
* Run backend project  
   backend is a standard spring boot project, you can run it in any java IDE
* Run frontend project  
  run with `npm run dev`
* visit webpage
  visit `http://localhost:8081`, frontend supports hot reload.             
 * CORS problem
    for convenien of development, we deploy frontend and backend separately, so the frontend supports hot reload. In this mode, frontend will request `localhost:8080` to fetch data, this will cause a CORS problem, so we add a configuration in `I18nConfig.java` to support CORS, this configuration will only be active under **develop** mode, please select the right maven profile to support this.

### Swagger supoort

Once deployed, you can check http://localhost:8080/swagger-ui.html to check all restful api and models


### License

Apache Dubbo ops is under the Apache 2.0 license, Version 2.0.
See [LICENSE](https://github.com/apache/incubator-dubbo-ops/blob/develop/LICENSE) for full license text.