# Dubbo ops

[![Build Status](https://travis-ci.org/apache/incubator-dubbo-ops.svg?branch=master)](https://travis-ci.org/apache/incubator-dubbo-ops)
[![codecov](https://codecov.io/gh/apache/incubator-dubbo-ops/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/incubator-dubbo-ops)
![license](https://img.shields.io/github/license/apache/incubator-dubbo-ops.svg)

### Screenshot

![index](https://raw.githubusercontent.com/apache/incubator-dubbo-ops/develop/dubbo-admin-frontend/src/assets/index.png)

### Frontend

- [Vue.js](https://vuejs.org) and [Vuetify](https://vuetifyjs.com)
- [dubbo-admin-frontend/README.md](dubbo-admin-frontend/README.md) for more detail

### Backend

> Configuration files (Before packaging application, make sure the correct profile in the MAVEN profiles was selected)
>
> - `application.properties`  
>   The generic configuration, it's permanent.
> - `application-develop.properties`  
>   The configuration for develop, it will be work when you use Maven's `develop` Profile.
> - `application-production.properties` (default)   
>   The configuration for production, it will be work when you use Maven's `production` Profile. Meanwhile, it's maven's default profile in this project.

### Build setup

1. Clone source code on develop branch
2. Specify registry address in `dubbo-admin-backend/src/resources/application-production.properties`
3. Build `mvn clean package`
4. Start `mvn --projects dubbo-admin-backend spring-boot:run`
5. Visit `http://localhost:8080`

### License

Apache Dubbo ops is under the Apache 2.0 license, Version 2.0.
See [LICENSE](https://github.com/apache/incubator-dubbo-ops/blob/develop/LICENSE) for full license text.
