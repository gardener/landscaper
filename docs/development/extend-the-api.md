# Extending the API

This document describes the steps that need to be performed when changing the API.

Generally, as Landscaper extends the Kubernetes API using CRD's, it follows the same API conventions and guidelines like Kubernetes itself.
[This document](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md) as well as [this document](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md) already provide a good overview and general explanation of the basic concepts behind it.
We are following the same approaches.

## Landscaper API

The Landscaper API is defined in `apis/{core,config}` directories and is the main point of interaction with the system.
The specific Deployer APIs are defined in `apis/deployers`.
It must be ensured that the API is always backwards-compatible.
If fields shall be removed permanently from the API then a proper deprecation period must be adhered to so that end-users have enough time adapt their clients.

All Landscaper and Deployer API definitions are provided as separate go submodule (see the apis [go.mod file](../../apis/go.mod) and an example of a multi module repo [here](https://github.com/go-modules-by-example/index/blob/master/009_submodules/README.md))
The separate repo was introduced so that external projects only use the apis module and do not have to care about the big dependency tree of the landscaper itself.

Using a submodule for the api means that the api module is a dependency of the main landscaper project. 
As a dependency, the module is vendored in `vendor/github.com/gardener/landscaper/apis` so after a module change you have to run `make revendor` in order to get the changes applied in the landscaper main project. (If you use `make generate` the revendoring is automatically done)


**Checklist** when changing the API:

1. Modify the field(s) in the respective Golang files of all external and the internal version.
    1. Make sure new fields are being added as "optional" fields, i.e., they are of pointer types, they have the `// +optional` comment, and they have the `omitempty` JSON tag.
    2. The Landscaper automatically generates the CRD's using CustomResource Definitions for each resource. So if your new type is custom resource, add the corresponding CR definition for the type and add it to the list of crds `ResourceDefinition` in [apis/core/{v1alpha1}/register.go](../../apis/core/v1alpha1/register.go#L69)
   ```go
   // InstallationDefinition defines the Installation resource CRD.
   var InstallationDefinition = lsschema.CustomResourceDefinition{
       Names: lsschema.CustomResourceDefinitionNames{
           Plural:   "installations",
           Singular: "installation",
           ShortNames: []string{
               "inst",
           },
           Kind: "Installation",
       },
       Scope:             lsschema.NamespaceScoped,
       Storage:           true,
       Served:            true,
       SubresourceStatus: true,
       AdditionalPrinterColumns: []lsschema.CustomResourceColumnDefinition{
           {
               Name:     "phase",
               Type:     "string",
               JSONPath: ".status.phase",
           },
           {
               Name:     "Execution",
               Type:     "string",
               JSONPath: ".status.executionRef.name",
           },
           {
               Name:     "Age",
               Type:     "date",
               JSONPath: ".metadata.creationTimestamp",
           },
       },
   }
   ```
1. If necessary then implement/adapt the conversion logic defined in the versioned APIs (e.g., `apis/core/v1alpha1/conversions.go`).
1. If necessary then implement/adapt defaulting logic defined in the versioned APIs (e.g., `apis/core/v1alpha1/defaults.go`).
1. Run the code generation: `make install-requirements generate`
1. If necessary then implement/adapt validation logic defined in the internal API (e.g., `apis/core/validation/validation.go`).
1. If necessary then adapt the exemplary YAML manifests of the resources defined in `example/*.yaml`.
1. In most cases it makes sense to add/adapt the documentation for administrators/operators and/or end-users in the `docs` folder to provide information on purpose and usage of the added/changed fields.
1. When opening the pull request then always add a release note so that end-users are becoming aware of the changes.

## Component configuration APIs

Most Landscaper components (controllers) have a component configuration that follows similar principles to the Gardener API.
Those component configurations are defined in `apis/config`.
Hence, the above checklist also applies for changes to those APIs.
However, since these APIs are only used internally and only during the deployment of Gardener the guidelines with respect to changes and backwards-compatibility are slightly relaxed.
If necessary then it is allowed to remove fields without a proper deprecation period if the release note uses the `action operator` keywords.

In addition to the above checklist:

1. If necessary then adapt the Helm chart of Gardener defined in `charts/gardener`. Adapt the `values.yaml` file as well as the manifest templates.
