# Dubbo admin

> dubbo admin front end and back end
![index](https://raw.githubusercontent.com/apache/incubator-dubbo-ops/develop/dubbo-admin-frontend/src/assets/index.png)
### front end
* Vue.js and Vuetify
* dubbo-admin-frontend/README.md for more detail

### back end
> Configuration files (Before packaging application, make sure the correct profile in the MAVEN profiles was selected)
> * `application.properties`   
>   The generic configuration
> * `application-test.properties`  
>   The configuration for test
> * `application-production.properties` (default)    
>   The configuration for production

## Build setup 

``` bash
# build
mvn clean install

# run
mvn --projects dubbo-admin-backend spring-boot:run

# visit
localhost:8080 

```
