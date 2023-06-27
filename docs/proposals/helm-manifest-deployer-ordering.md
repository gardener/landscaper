# Ordering of Objects during deployment and deletion of Charts with Helm Manifest Deployer

If deploying a DeployItem with the Helm Deployer and the entry `helmDeployment` is set to false the Chart is only 
templated with Helm and the resulting manifests are just deployed with standard kubernetes means instead of using Helm
([see](https://github.com/gardener/landscaper/blob/master/docs/deployer/helm.md#manifest-only-deployment)).

This document describes the behaviour of the current implementation and proposals how to improve the logic.  

## Installation and Upgrade of Chart

During the installation and upgrade the manifests are currently deployed in the following order:

- CRDs
- Manifests for non namespaced objects like namespaces, cluster roles etc.
- Manifests for namespaced objects, i.e. objects stored in a namespace

This behaviour seems to be quite reasonable and will not be changed.

Open questions: 

- Do we require some possibility to influence the deploy order more fine grained?
- Do we need a more elaborated order like helm ([see](https://helm.sh/docs/intro/using_helm/))?

## Uninstall Chart

### Current Status

Currently, when removing a Chart, the objects deployed by the chart are deleted in the following order:

- Namespaced objects deployed by the Chart
- Not namespaced objects deployed by the Chart
- CRDs deployed by the Chart

The algorithm does not wait until particular objects are gone before it continues deleting the next ones.The removal of 
a Chart is successful if all objects were gone. Objects not deployed by the Chart, e.g. custom resources (CRs) deployed
by some operator/job are not removed.

### New Solution

## Default Deletion Behaviour

We propose the following solution to have more control over the deletion process.

The basic order of deleting the deployed manifests remains the same as before and is divided into three deletion groups:

- Namespaced objects deployed by the Chart
- Not namespaced objects deployed by the Chart (except CRDs)
- CRDs deployed by the Chart

The deletion continues only with the next object/deletion group if all objects from the groups before are gone. This is 
different to the current approach. The deletion is tried as long until all objects of all 3 deletion groups are gone
or the specified configurable timeout (default 5 min.) comes into the game and the deletion failed.

## Custom Deletion Behaviour

### Basics

To change the deletion behaviour of a DeployItem, you could specify your own custom deletion groups. A deletion group describes
a set of k8s objects which should be deleted. The deletion groups are defined as a list and one deletion group after 
the other is processed. Thereby again, all objects of a deletion group must be gone before the deletion of objects of the next 
deletion group starts. This is tried until all objects of all deletion groups are gone (SUCCESS) or the timeout comes 
into place (FAILED). If you have specified your own deletion groups, the default deletion behaviour is completely disabled.

Custom deletion groups are defined in the section `deletionGroups` as shown below:

```yaml
deployItems:
  - name: my-deploy-item
    type: landscaper.gardener.cloud/helm
    config:
      apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
      kind: ProviderConfiguration
      ...
      deletionGroups:
      - <deletionGroup-1>
      - <deletionGroup-2>
      ...
```

### Predefined Resources

The most simple specification for a deletion group is using predefined resource groups. The following example
shows how to specify the default deletion behaviour with 3 such deletion groups:


```yaml
deployItems:
  - name: my-deploy-item
    type: landscaper.gardener.cloud/helm
    config:
      apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
      kind: ProviderConfiguration
      ...
      deletionGroups:
        - predefinedResourceGroup: 
            type: namespaced-resources
        - predefinedResourceGroup:
            type: cluster-scoped-resources     # does not include the crds
        - predefinedResourceGroup: 
            type: crds
```

Note that you can omit the section `deletionGroups` only if you accept the exact default behaviour.
As soon as you want to deviate from the default behaviour, you have to specify the `deletionGroups` list with all
items you want to be processed.

##### Example: skip deletion of CRDs

In this example, the deletion of CRDs is skipped:

```yaml
deletionGroups:
  - predefinedResourceGroup: 
      type: namespaced-resources
  - predefinedResourceGroup: 
      type: cluster-scoped-resources
```

### Specific Resources

If you want to specify a deletion group specifying the deletion of particular resource types you could use the
following syntax where you specify the types of the objects which should be deleted in that deletion group:

```yaml
deletionGroups:
  - resources:
      - group:   ...
        version: ...
        kind:    ...
      - group:   ...
        version: ...
        kind:    ...
    ...
```

##### Example: delete certain resources first

In this example, the ConfigMaps and Secrets of the chart are deleted first. Only when all of these have gone, the
other resources will be deleted as usual.

```yaml
deletionGroups:
  - resources:
      - group:   ""
        version: "v1"
        kind:    "configmaps"
      - group:   ""
        version: "v1"
        kind:    "secrets"
  - predefinedResourceGroup: 
      type: namespaced-resources
  - predefinedResourceGroup: 
      type: cluster-scoped-resources
  - predefinedResourceGroup: 
      type: crds
```

Every item of the list `deletionGroups` must contain exactly one of `resources` or `predefinedResourceGroup` to
define a set of resources.

### Force Delete

Another important point is the possibility to force the deletion of particular objects by specifying
the entry `forceDelete`. The meaning of the additional fields is that after a successful deletion call to all objects 
of the deletion group, the finalizer of these objects are also removed.

##### Example: force-delete

In this example, the `force-delete` mode is enabled for all namespaced and cluster-scoped resources:

```yaml
deletionGroups:
  - predefinedResourceGroup: namespaced-resources
    force-delete:
      enabled: true
  - predefinedResourceGroup: cluster-scoped-resources
    force-delete:
      enabled: true
  - predefinedResourceGroup: crds
```

Of course `forceDelete` could also be applied for cluster group definitions for particular object types.

### Deleting all Resources

In the current deletion process only objects deployed by the chart are removed. This is also the default behaviour
for the new approach. 

You can change this behaviour by specifying `seletor.all=true` and all objects of that type are removed. We could later 
extend the selector by rules for namespaces, labels, object names etc. to allow more elaborated deletion rules.

##### Example: delete resources outside the chart

In this example, all custom resources of a certain group-version-kind are deleted in the beginning. Because of the
selector `all: true`, all resources of that group-version-kind are deleted, regardless whether they were deployed by
the chart or not.

```yaml
deletionGroups:
  - resources:
      - group:   "my.group"
        version: "v1"
        kind:    "mycustomresources"
        selector:
          all: true
  - predefinedResourceGroup: namespaced-resources
  - predefinedResourceGroup: cluster-scoped-resources
  - predefinedResourceGroup: crds
```

##### General Structure of deletion groups

The general syntax of deletion groups is:

```yaml
deletionGroups:
  - predefinedResourceGroup: 
      type: ( "namespaced-resources" | "cluster-scoped-resources" | "crds" )
    resources:
      - group:   ...
        version: ...
        kind:    ...
        selector:
          all: true
    force-delete:
      enabled: true
```

The field `deletionGroups` is a list. Its items have the following fields:

- **predefinedResourceGroup:** this field is optional, but exactly one of `resources` or `predefinedResourceGroup` must
  be set. The field has type field with the allowed values:
    - `namespaced-resources`
    - `cluster-scoped-resources`
    - `crds`

- **resources:** this field is optional, but exactly one of `resources` or `predefinedResourceGroup` must be set.
  The field is a list. Each item must have fields `group`, `version`, `kind` to specify a type of resources.
  Optionally, a `selector` can be specified; currently, only the selector `all: true` is supported to indicate that
  resources should be deleted even if they were not deployed by the chart.

- **force-delete:** this field is optional. It is an object with field `enabled: (true | false )`.



> Maybe later we need a field `excludedResources` to express something like: all namespaced resources except configmaps.


## Open questions: 

- How to handle deletions in Chart upgrades?