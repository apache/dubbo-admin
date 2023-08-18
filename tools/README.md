# Use the tools tool to quickly add dubbo resources

- define a crd

- Generate the crd definition in the form of proto

  For example:

  This is the crd definition of ServiceNameMapping:

   ```yaml
   apiVersion: apiextensions.k8s.io/v1
   kind: CustomResourceDefinition
   metadata:
     name: servicenamemappings.dubbo.apache.org
   spec:
     group: dubbo.apache.org
     versions:
     - name: v1alpha1
       served: true
       storage: true
       schema:
         openAPIV3Schema:
           properties:
             spec:
               description:
                 'Spec defines the behavior of a service mapping.
                  https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status'
               properties:
                 applicationNames:
                   items:
                     type: string
                   type: array
                 interfaceName:
                   type: string
               type: object
           type: object
     scope: Namespaced
     names:
       plural: servicenamemappings
       singular: servicenamemapping
       kind: ServiceNameMapping
       shortNames:
         - snp
  
   ```

  It is correspondingly written as pb:

   ```protobuf
   syntax = "proto3";
  
   package dubbo.apache.org.v1alpha1;
  
   option java_multiple_files = true;
  
   message ServiceNameMappingToClient {
     string key = 1;
     ServiceNameMapping spec = 2;
   }
  
   message ServiceNameMapping {
     string interfaceName = 1;
     repeated string applicationNames = 2;
   }
   ```

  Use `protoc-gen-gogo` to generate it under `api/resource/v1alpha1`

- Go to the `tools/resource-gen` directory and write metadata.yaml. `dds: true` means that this resource will be included in the push scope. validate: Represents the validation function. Each resource can have a different validation function. If you don't need a validation function, just fill in `EmptyValidate`. Validation functions are written in `validation.go` of `pkg/core/validation`.

- ```sh
   cd tools
   chmod a+x gen.sh
   ./gen.sh
   ```

- You can see that something is being generated under `pkg/core/gen` at this point. At this point, you can use the code-generator officially provided by k8s to generate the corresponding generated through types.go, **note that it must be named generated**.

  > Why do you need code-generator?
  >
  > The reason is simple, we have no way of knowing where your code-generator's code warehouse is installed.

- Write `pkg/dds/storage/generate.go`, refer to other resources.