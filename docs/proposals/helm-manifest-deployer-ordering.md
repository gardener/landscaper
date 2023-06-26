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

---

### Proposal for an Enhanced Deletion Logic

We propose to enhance the deletion logic with the following features:

- **Subdivision into groups:** the set of all resources to be deleted can be divided into groups.
An order can be defined in which these groups are handled.
Landscaper waits until all objects of a group have gone before it deletes the objects of the next group.  
- **Force-delete:** a force-delete mode can be enabled on group level. If force-delete is enabled for a group, 
its objects are not only deleted, but also their finalizers are directly removed.  
- **Deleting objects outside the chart:** it is possible to delete resources that were not deployed by the Helm chart.
For example, a Helm chart might deploy an "operator" which then in turn creates certain custom resources. When the 
operator is deleted one might also delete these custom resource, even if they are not part of the Helm chart.

By default, there are three groups, handled in the following order:

1. The group of namespaced resources of the chart  
2. The group of cluster-scoped resources of the chart (without CRDs)  
3. The CRDs of the chart  

The groups can (optionally) be defined in a section `deletionGroups` of DeployItems.
We describe the syntax of the `deletionGroups` in a sequence of examples, before we give the general scheme.

##### Example: default behaviour

If you are happy with the above default behaviour, you need not specify the groups at all. However, the section
`deletionGroups` in the DeployItem below would have the same effect:

```yaml
deployItems:
  - name: my-deploy-item
    type: landscaper.gardener.cloud/helm
    config:
      apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
      kind: ProviderConfiguration
      ...
      deletionGroups:
        - predefinedResourceGroup: namespaced-resources
        - predefinedResourceGroup: cluster-scoped-resources     # does not include the crds
        - predefinedResourceGroup: crds
```

Note that you can omit the section `deletionGroups` only if you accept the exact default behaviour. 
As soon as you want to deviate from the default behaviour, you have to specify the `deletionGroups` list with all
items you want to be processed.


##### Example: skip deletion of CRDs

In this example, the deletion of CRDs is skipped:

```yaml
deletionGroups:
  - predefinedResourceGroup: namespaced-resources
  - predefinedResourceGroup: cluster-scoped-resources
```

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

##### Example: delete certain resources first

In this example, the ConfigMaps and Secrets of the chart are deleted first. Only when all of these have gone, the
other resources will be deleted as usual.

Every item of the list `deletionGroups` must contain exactly one of `resources` or `predefinedResourceGroup` to
define a set of resources.

```yaml
deletionGroups:
  - resources:
      - group:   ""
        version: "v1"
        kind:    "configmaps"
      - group:   ""
        version: "v1"
        kind:    "secrets"
  - predefinedResourceGroup: namespaced-resources
  - predefinedResourceGroup: cluster-scoped-resources
  - predefinedResourceGroup: crds
```

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

```yaml
deletionGroups:
  - predefinedResourceGroup: ( "namespaced-resources" | "cluster-scoped-resources" | "crds" )
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
be set. The field has type string. The allowed values are:  
  - `namespaced-resources`  
  - `cluster-scoped-resources`  
  - `crds`  

- **resources:** this field is optional, but exactly one of `resources` or `predefinedResourceGroup` must be set.
The field is a list. Each item must have fields `group`, `version`, `kind` to specify a type of resources. 
Optionally, a `selector` can be specified; currently, only the selector `all: true` is supported to indicate that
resources should be deleted even if they were not deployed by the chart.  

- **force-delete:** this field is optional. It is an object with field `enabled: (true | false )`.  



> Maybe we need a field `excludedResources` to express something like: all namespaced resources except configmaps.


---

### New Solution

We propose the following solution to have more control over the deletion process.

The basic order of deleting the deployed manifests remains the same as before (deletionRank is specified below):

- Namespaced objects deployed by the Chart (deletionRank=100)
- Not namespaced objects deployed by the Chart (except CRDs) (deletionRank=200)
- CRDs deployed by the Chart (deletionRank=300)

The deletion continues only with the next object group if all objects from the groups before are gone. This is different
to the current approach.

Each of these standard groups of objects have a particular deletionRank which determines the deletion order. To change 
the deletion order of particular resources you could overwrite their default deletionRank in the specification of a 
DeployItem as shown next:

```
deployItem:
...
  - deletionOrderRules:
    - deletionRank: <some integer number>
      types:
        - group: ...
          version: ...
          kind: ... 
```

By such a deletion order rule, the objects specified get the new deletionRank of the rule, and are deleted after all 
other objects with a lower deletionRank but before those with a higher deletionRank. Thereby, the deletion of objects 
with higher rankings is only continued if all objects with a deletionRank lower or equal to the deletionRank, specified 
in the rule, are gone.

In the current deletion process only objects deployed by the chart are removed. If you specify `seletor.all=true`
all objects of that type are removed. We could later extend the selector by rules for namespaces, labels, object names 
etc. 

```
deployItem:
...
  - deletionOrderRules:
    - deletionRank: <some integer number>
      types:
        - group: ...
          version: ...
          kind: ...
          selector: # optional
            all: true 
```

Another important point is the possibility to force the deletion of particular objects by specifying
the entry `forceDelete`.

```
deployItem:
...
  - deletionOrderRules:
    - deletionRank: <some integer number>
      types:
        - group: ...
      forceDelete: # optional
        enabled: <true/false>
```

The meaning of the additional fields is that after a successful deletion call to all objects of the deletionRank, 
the finalizer of these objects are also removed.

Of course, it is also possible to specify forced cleanup for the standard groups, e.g. the following
example introduces such a rule for the CRs of CRDs specified in the Chart, which have the default deletionRank of 100
200 or 300. You see that the types section is not allowed and therefore skipped in this case. 

```
deployItem:
...
  - deletionOrderRules:
    - deletionRank: 100
      forceDelete: # optional
        enabled: <true/false>
```

If you do not specify a deletion rule for the default deletionRank of 100, 200 or 300, default deletion rules are 
automatically added for the missing ones, which have the following form with the meaning that all objects of the 
included type deployed by the chart has to be deleted:

```
deployItem:
...
  - deletionOrderRules:
    - deletionRank: <100, 200, or 300>
```

It is required that all rules must have a different deletionRank!

#### How is a set of deletionOrderRules processed?

Assume you have the rules R1 to Rn. This set also includes all default deletion rules. The rules are sorted increasingly
according to their deletionRank. Then one rule after the other is processed and the respective objects are deleted.

#### Example

The following example specifies the following rules:

- First delete all config maps, including those not deployed by the chart
- Next delete all secrets including their finalizer deployed by the chart
- Next delete all CRs of group/version/kind=g1/v1/k1
- Next delete all CRs deployed by the chart of group/version/kind=g2/v2/k2
- Next delete all namespaced objects deployed 
- Next delete all namespaces deployed by the chart
- Next delete all not namespaced objects deployed by the chart
- Next delete CRDs deployed by the chart

```
deployItem:
...
  - deletionOrderRules:
  
    - deletionRank: 50 
      types:
      - group: ""
        version: v1
        kind: configmap
        selector: 
          all: true 
          
    - deletionRank: 51
      types:
      - group: ""
        version: v1
        kind: secret
      forceDelete: 
        enabled: true
        
    - deletionRank: 52
      types:
      - group: g1
        version: v1
        kind: k1
        selector: 
          all: true 
          
    - deletionRank: 53
      types:
      - group: g2
        version: v2
        kind: k3
        
    - deletionRank: 150 # rank higher than the namespaced objects
      types:
      - group: 
        version: v1
        kind: namespace

```

Open questions: 
- How to handle different CRs versions of one CRD?
- How to handle deletions in Chart upgrades?