### dubbo-ops
The following modules in [dubbo](https://github.com/alibaba/dubbo) have been moved here:

* dubbo-admin
* dubbo-monitor-simple
* dubbo-registry-simple

### How to use it
You can get a release of dubbo monitor in two steps:

- Step 1: 
```
git clone https://github.com/apache/incubator-dubbo-ops
```

- Step 2: 
```
cd incubator-dubbo-ops && mvn package
```

Then you will find:
- dubbo-admin-2.0.0.war in incubator-dubbo-ops\dubbo-admin\target directory.You can deploy it into your application server.
- dubbo-monitor-simple-2.0.0-assembly.tar.gz in incubator-dubbo-ops\dubbo-monitor-simple\target directory. Unzip it you will find the shell scripts for starting or stopping monitor.
- dubbo-registry-simple-2.0.0-assembly.tar.gz in incubator-dubbo-ops\dubbo-registry-simple\target directory. Unzip it you will find the shell scripts for starting or stopping registry.


