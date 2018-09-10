# dubbo-ops

[![Build Status](https://travis-ci.org/apache/incubator-dubbo-ops.svg?branch=master)](https://travis-ci.org/apache/incubator-dubbo-ops) 
[![Gitter](https://badges.gitter.im/alibaba/dubbo.svg)](https://gitter.im/alibaba/dubbo?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

The following modules in [Apache Dubbo(incubating)](https://github.com/apache/incubator-dubbo) have been moved here:

* dubbo-admin
* dubbo-monitor-simple
* dubbo-registry-simple


## NOTICE  
dubbo admin is under refactoring, please checkout the `develop` branch


## How to use it

### dubbo admin

Dubbo admin is a spring boot application, you can start it with fat jar or in IDE directly.

### dubbo monitor and dubbo registry

You can get a release of dubbo monitor in two steps:

* Step 1:
```
git clone https://github.com/apache/incubator-dubbo-ops
```

* Step 2:
```
cd incubator-dubbo-ops && mvn package
```

Then you will find:

* dubbo-monitor-simple-2.0.0-assembly.tar.gz in incubator-dubbo-ops\dubbo-monitor-simple\target directory. Unzip it you will find the shell scripts for starting or stopping monitor.
* dubbo-registry-simple-2.0.0-assembly.tar.gz in incubator-dubbo-ops\dubbo-registry-simple\target directory. Unzip it you will find the shell scripts for starting or stopping registry.
