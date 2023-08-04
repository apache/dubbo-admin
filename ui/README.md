# dubbo-admin-ui

> dubbo admin ui based on vuetify
> standard front end project

## Build Setup

前端依赖的所有后端 Swagger API 文档目前存放在 `[hack/swagger/swagger.json](hack/swagger/swagger.json)` 目录。

### 如何进行前端开发

### 前端打包
如果你有修改前端代码，则需要切换到 `dubbo-admin-ui` 目录后，按照以下方式重新打包前端代码后并重新启动 Admin 进程。

1. 执行以下命令安装相关依赖包
```shell
yarn
```

2. 构建前端组件

```shell
yarn build
```

3. 拷贝构建结果到 admin 发布包

```shell
rm -rf ../app/dubbo-ui/dist
cp -R ./dist ../app/dubbo-ui/
```


For a detailed explanation on how things work, check out the [guide](http://vuejs-templates.github.io/webpack/) and [docs for vue-loader](http://vuejs.github.io/vue-loader).

managed by [front end maven plugin](https://github.com/eirslett/frontend-maven-plugin)

