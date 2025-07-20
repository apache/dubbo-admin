<h1 align="center">
Dubbo Admin
</h1>

[![Build](https://github.com/apache/dubbo-kubernetes/actions/workflows/ci.yml/badge.svg)](https://github.com/apache/dubbo-kubernetes/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/apache/dubbo-kubernetes/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-kubernetes)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

Dubbo Admin is the console designed for better visualization of Dubbo services, providing tools for building and 
deploying Dubbo applications in various environments, including Kubernetes and VM.

## Repositories
The main code repositories of Dubbo Admin include:

- [dubboctl](./dubboctl). This directory contains code for the command line tool.
- [ui-vue3](./ui-vue3). This directory contains code for the front-end.
- [app](./app), [pkg](./pkg). These directories contains code for the
[dubbo admin console](https://github.com/apache/dubbo-admin/blob/develop/app/README.md).

## Quick Start
Please refer to [official website](https://cn.dubbo.apache.org/zh-cn/overview/home/).

## Roadmap
Please refer to [RoadMap](https://github.com/apache/dubbo-admin/discussions/1300).

## Contributing

### Admin UI
- [Vue3](https://vuejs.org/) and [Vite](https://vite.dev/)
- [ui-vue3/README.md](https://github.com/apache/dubbo-admin/ui-vue3/README.md) for more detail

### Admin Server
- Golang, Gin, Kubernetes
- [docs/server-develop](https://github.com/apache/dubbo-admin/docs/server-develop/README.md) for more detail

### Other Information 
Refer to [CONTRIBUTING.md](https://github.com/apache/dubbo-kubernetes/blob/master/CONTRIBUTING.md)

## License
Apache License 2.0, see [LICENSE](https://github.com/apache/dubbo-kubernetes/blob/master/LICENSE).