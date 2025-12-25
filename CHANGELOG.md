# 0.7.0 [Milestone]
## Summary
This release is a milestone version. The entire backend system is completely rewritten using Go, meanwhile, 
the frontend ui is redesigned and completely rewritten using Vue3. 
The whole refactor extends the base functions of past system and introduces many new features including the native support of kubernetes, 
deep integration of observability system and form-based traffic rule publishing.


## New Features
- Support resource-level isolation of store. [#1310](https://github.com/apache/dubbo-admin/pull/1310)
- Introduce informer framework to support discovery and engine. [#1314](https://github.com/apache/dubbo-admin/pull/1314)
- Support memory type of store based on cache in client-go. [#1332](https://github.com/apache/dubbo-admin/pull/1332)
- Kubernetes implemention of Engine. [#1340](https://github.com/apache/dubbo-admin/pull/1340)
- Implement Counter in local cache. [#1345](https://github.com/apache/dubbo-admin/pull/1345)
- Introduce a unified error and display error in a better way. [#1353](https://github.com/apache/dubbo-admin/pull/1353), [#1355](https://github.com/apache/dubbo-admin/pull/1355)
- Support multiple registries in console. [#1356](https://github.com/apache/dubbo-admin/pull/1356)
- Nacos Implemention of Discovery. [#1367](https://github.com/apache/dubbo-admin/pull/1367)
- Support components to start in dependency order. [#1370](https://github.com/apache/dubbo-admin/pull/1370)
- Zookeeper Implemention of Discovery. [#1371](https://github.com/apache/dubbo-admin/pull/1371)

## Bug Fixes
- Fix compile error. [#1325](https://github.com/apache/dubbo-admin/pull/1325)
- Fix counter registration bug. [#1369](https://github.com/apache/dubbo-admin/pull/1369)
- Fix npe caused by incorrect indexer initiation order. [#1372](https://github.com/apache/dubbo-admin/pull/1372)
- Redirect to login page when user is not logged in. [#1373](https://github.com/apache/dubbo-admin/pull/1373)

## Enhancements
- Migrate the base project from [dubbo-kubernetes](https://github.com/apache/dubbo-kubernetes) to this repository. [#1302](https://github.com/apache/dubbo-admin/pull/1302)
- Reorganize the project structure and tidy up the legacy code. [#1304](https://github.com/apache/dubbo-admin/pull/1304), [#1307](https://github.com/apache/dubbo-admin/pull/1307), [#1311](https://github.com/apache/dubbo-admin/pull/1311)

## Documentation & CI
- Fix ci problems. [#1320](https://github.com/apache/dubbo-admin/pull/1320), [#1322](https://github.com/apache/dubbo-admin/pull/1322), [#1326](https://github.com/apache/dubbo-admin/pull/1326)

## Contributors
Thanks to all contributors for their efforts in this release: 

[AlexStocks](https://github.com/AlexStocks), [ikun-Lg](https://github.com/ikun-Lg),[everfid-ever](https://github.com/everfid-ever), 
[WyRainBow](https://github.com/WyRainBow), [Helltab](https://github.com/Helltab),[stringl1l1l1l](https://github.com/stringl1l1l1l), 
[marseilspirit](https://github.com/marsevilspirit), [mfordjody](https://github.com/mfordjody), [tew-axiom](https://github.com/tew-axiom), 
[robocanic](https://github.com/robocanic)

