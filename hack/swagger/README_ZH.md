# swaggo 使用说明文档

https://github.com/swaggo/swag/blob/master/README_zh-CN.md

# 生成 dubbo-admin api swagger 文档

在项目根目录下执行 make help

```shell
make help

dubbo-admin-swagger-gen  Generate dubbo-admin swagger docs.
dubbo-admin-swagger-ui  Generate dubbo-admin swagger docs and start swagger ui.

```

1. 生成 dubbo-admin swagger 文件

执行 make dubbo-admin-swagger-gen

在 hack/swagger/docs 目录下生成 dubbo-admin swagger 三个文件: docs.go, swagger.json, swagger.yaml

```shell
.
├── README_ZH.md
├── docs
│ ├── docs.go
│ ├── swagger.json
│ └── swagger.yaml
├── go.mod
├── go.sum
└── main.go

```

2.  生成 dubbo-admin swagger 文件同时并且启动本地 web server 查看 swagger ui

执行 make dubbo-admin-swagger-ui

通过 http://127.0.0.1:38081/swagger/index.html 查看 swagger ui.


