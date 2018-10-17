# Dubbo admin

> dubbo admin front end and back end
![index](https://raw.githubusercontent.com/apache/incubator-dubbo-ops/develop/dubbo-admin-frontend/src/assets/index.png)
### front end
* Vue.js and Vuetify
* dubbo-admin-frontend/README.md for more detail

### back end
> Configuration files (Before packaging application, make sure the correct profile in the MAVEN profiles was selected)
> * `application.properties`   
>   The generic configuration, it's permanent.
> * `application-test.properties`  
>   The configuration for test, it will be work when you use Maven's `develop` Profile.
> * `application-production.properties` (default)    
>   The configuration for production, it will be work when you use Maven's `production` Profile. Meanwhile, it's maven's default profile in this project.

## Build setup 

``` bash
# build
mvn clean install

# run
mvn --projects dubbo-admin-backend spring-boot:run

# visit
localhost:8080 

```
