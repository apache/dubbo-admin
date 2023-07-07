# Dubbo-Admin Helm Chart

* Apache [Dubbo-Admin](https://cn.dubbo.apache.org/) is a console for better visualization of Dubbo services, it provides fully support for Dubbo and is compatible with 2.7.x, 2.6.x and 2.5.x.


## Installing the Chart

To install the my-release deployment:

```shell
helm install my-release dubbo-admin
```

## Uninstalling the Chart

To uninstall/delete the my-release deployment:

```shell
helm delete my-release
```

## Configuration

| Parameter                  | Description                                                                                                         | Default |
|----------------------------|---------------------------------------------------------------------------------------------------------------------|---------|
| global.imageRegistry       | The global configuration option for specifying the image registry address.                                          | ""      |
| global.imagePullSecrets    | Specify mirror pull credentials. Used to authenticate and authorize access to private container image repositories. | []      |
| rbac.enabled               | Used to enable or disable role-based access control.                                                                | true    |
| rbac.pspEnabled            | Used to enable or disable Pod Security Policies.                                                                    | true    |
| replicas                   | Used to specify the number of copies to be deployed.                                                                | 1       |
| labels                     | The label used to specify the deployment resource.                                                                  | {}      |
| annotations                | Comments for specifying deployment resources.                                                                       | {}      |
| nodeSelector               | Used to specify a node selector to select a specific node to run the deployed instance.                             | {}      |
| affinity                   | Used to specify affinity rules to control the scheduling and distribution of resources in the cluster.              | {}      |
| tolerations                | Used to specify tolerance rules to tolerate specific node taint.                                                    | {}      |
| serviceAccount.enabled     | Used to enable or disable ServiceAccount.                                                                           | true    |
| serviceAccount.name        | Used to specify the name of the ServiceAccount.                                                                     | {}      |
| serviceAccount.nameTest    | Used to specify the name of another ServiceAccount.                                                                 | {}      |
| serviceAccount.labels      | The label used to specify the ServiceAccount.                                                                       | {}      |
| serviceAccount.annotations | The comment used to specify the ServiceAccount.                                                                     | {}      |
| imagePullSecrets           | Configuration option for specifying mirror pull credentials.                                                        | []      |