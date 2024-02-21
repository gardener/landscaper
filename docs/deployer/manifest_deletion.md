---
title: Deletion of Manifest and Manifest-Only Helm DeployItems
sidebar_position: 6
---

# Deletion of Manifest and Manifest-Only Helm DeployItems

This chapter is only relevant for:
- the [manifest deployer](./manifest.md),
- and [manifest-only deployments](./helm.md#manifest-only-deployment) with the helm deployer, i.e. if you use the helm
  deployer with the setting `helmDeployment: false` in the provider configuration of the DeployItem.

## Default Deletion Behaviour

When a DeployItem is deleted, the deployed resources are divided into three groups, so-called *deletion groups*:

1. Namespaced resources,
2. Cluster-scoped resources (except custom resource definitions),
3. Custom resource definitions.

The resources are deleted group by group, i.e. first the namespaced resources, next the cluster-scoped resources, and
finally the CRDs. After the resources of a group have been deleted, the algorithm waits until these resources are gone 
before it proceeds with the next group.

The deletion is tried until all resources of the deletion groups are gone, or the timeout of the DeployItem is reached 
and the deletion fails.

## Custom Deletion Behaviour

You can customize the deletion behaviour to control which objects are deleted and in which order.
For this purpose you can specify a list of deletion groups in the provider configuration of the
DeployItem like this:

```yaml
deployItems:
  - name: my-deploy-item
    type: landscaper.gardener.cloud/helm
    config:
      apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
      kind: ProviderConfiguration
      ...
      deletionGroups:
      - <deletion group 1>
      - <deletion group 2>
      ...
```

Each deletion group describes a set of resources which should be deleted. As in the default behaviour, the resources are
deleted group by group, and the algorithm waits until the resources of a group are gone before proceeding with the next 
group.

The list of deletion groups can be build from [predefined resource groups](#predefined-resource-groups) and 
[custom resource groups](#custom-resource-groups).

Note that you can omit the section `deletionGroups` only if you accept the exact default behaviour.
As soon as you want to deviate from the default behaviour, you have to specify the `deletionGroups` list with all
groups you want to be processed.

### Predefined Resource Groups

The simplest way to specify a deletion group is using one of the following predefined groups: 

- the group of namespaced resources, 
- the group of cluster-scoped resources (except custom resource definitions),
- the group of custom resource definitions,
- the empty group.

The following example shows how one could imitate the default behaviour with three predefined groups:

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
            type: cluster-scoped-resources   # does not include the crds
        - predefinedResourceGroup: 
            type: crds
```

#### Example: skip deletion of CRDs

In this example, the deletion of CRDs is skipped:

```yaml
deletionGroups:
  - predefinedResourceGroup:
      type: namespaced-resources
  - predefinedResourceGroup:
      type: cluster-scoped-resources
```

#### Example: deleting no resources

There is a predefined empty group. It allows you to express that no resources should be deleted:

```yaml
deletionGroups:
  - predefinedResourceGroup: 
      type: empty
```

Note: if you specify no deletion groups, the [default deletion behaviour](#default-deletion-behaviour) would be 
applied instead.

### Custom Resource Groups

You can define a deletion group which consists of resources of particular `apiVersion`s and `kind`s. The syntax is:

```yaml
deletionGroups:
  - customResourceGroup:
      resources:
        - apiVersion: ...
          kind:       ...
        - apiVersion: ...
          kind:       ...
```

Optionally, you can further restrict the set of resources with filters for names, namespaces, or both:

```yaml
deletionGroups:
  - customResourceGroup:
      resources:
        - apiVersion: ...
          kind:       ...
          names:
            - name1
            - name2
          namespaces:
            - namespace1
            - namespace2
```

#### Example: delete certain resources first

In this example, the ConfigMaps and Secrets are deleted first. Only when all of them are gone, the
other namespaced resources will be deleted as usual. Next, the namespaces are deleted, followed by the
cluster-scoped resources, and finally the CDRs.

```yaml
deletionGroups:
  - customResourceGroup:
      resources:
        - apiVersion: "v1"
          kind: "configmaps"
        - apiVersion: "v1"
          kind: "secrets"
  - predefinedResourceGroup: 
      type: namespaced-resources
  - customResourceGroup:
      resources:
        - apiVersion: "v1"
          kind: "namespaces"
  - predefinedResourceGroup: 
      type: cluster-scoped-resources
  - predefinedResourceGroup: 
      type: crds
```

Every item of the list `deletionGroups` must contain exactly one of `predefinedResourceGroup` or `customResourceGroup`
to define a set of resources.

#### Example: delete CRs first

In this example, some custom resources are deleted before the namespaced resources. As the deletion algorithm only 
proceeds with the namespaced resources after all custom resources of the first group are gone, a potential operator 
has the time to do their cleanup, before they remove the finalizers from the custom resources.

```yaml
deletionGroups:
  - customResourceGroup:
      resources:
        - apiVersion: <apiVersion of CR>"
          kind:       <kind of CR>
  - predefinedResourceGroup: 
      type: namespaced-resources
  - predefinedResourceGroup: 
      type: cluster-scoped-resources
  - predefinedResourceGroup: 
      type: crds
```

### Force Delete

You can force the deletion of resources by adding the field `forceDelete: true` to a deletion group. This means that
after a successful deletion call to all resources of the group (i.e. after adding a deletion timestamp) 
the finalizers of the resources are also removed. The setting `forceDelete: true` is supported in predefined and 
custom resource groups.

#### Example: forceDelete

In this example, the `forceDelete` mode is applied to ConfigMaps and cluster-scoped resources:

```yaml
deletionGroups:
  - customResourceGroup:
      resources:
        - apiVersion: v1
          kind: configmaps
      forceDelete: true
  - predefinedResourceGroup: 
      type: namespaced-resources
  - predefinedResourceGroup: 
      type: cluster-scoped-resources
      forceDelete: true
  - predefinedResourceGroup: 
      type: crds
```

### Deleting all resources

By default, the deletion process only deletes resources that were deployed by the DeployItem. 

You can change this behaviour by adding the field `deleteAllResources: true` to a custom resource group. 
In this case all resources that are specified in the group will be deleted, regardless whether they were deployed
by the DeployItem or not.

This approach allows to delete also resources which where not directly created by a Helm chart, but for example by an 
operators or jobs which itself was deployed by the chart.

:warning: Use this feature with care! There is no further check to prevent you from deleting more than intended.

#### Example: delete resources outside a chart

In this example, all custom resources of a certain apiVersion and kind are deleted in the beginning. Because of the
field `deleteAllResources: true`, all resources of that apiVersion and kind are deleted, even those which were not 
deployed by the chart.

```yaml
deletionGroups:
  - customResourceGroup:
      resources:
        - apiVersion: v1
          kind: mycustomresource
      deleteAllResources: true
  - predefinedResourceGroup: 
      type: namespaced-resources
  - predefinedResourceGroup: 
      type: cluster-scoped-resources
  - predefinedResourceGroup: 
      type: crds
```

## Deletion Behaviour During Update

During an update of a DeployItem, resources which belonged to the old version, but no longer to the new version,
will be deleted. This deletion process during an update follows the same rules. There is the same
[default deletion behaviour](#default-deletion-behaviour), and alternatively you can customize the behaviour via
`deletionGroupsDuringUpdate`:

```yaml
deployItems:
  - name: my-deploy-item
    type: landscaper.gardener.cloud/helm
    config:
      apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
      kind: ProviderConfiguration
      ...
      deletionGroupsDuringUpdate:
        - <deletion group during update 1>
        - <deletion group during update 2>
      ...
      deletionGroups:
        - <deletion group 1>
        - <deletion group 2>
      ...
```

Note that the behaviour during update and delete is controlled by different lists of deletion groups:
`deletionGroupsDuringUpdate`, resp. `deletionGroups`. The distinction is necessary in some scenarios. For example, 
the setting `deleteAllResources: true` could be desirable during a deletion, but disastrous during an update.

## General structure of deletion groups

The general syntax of deletion groups (resp. deletion groups during update) is:

```yaml
deletionGroups:
  - predefinedResourceGroup: 
      type: ("namespaced-resources" | "cluster-scoped-resources" | "crds" | "empty")
      forceDelete: (true | false)
  - customResourceGroup:
      resources:
        - apiVersion: ...
          kind:       ...
          names:
            - name1
            - name2
          namespace:
            - namespace1
            - namespace2
      forceDelete: (true | false)
      deleteAllResources: (true | false)
```

#### Deletion groups

The provider configuration of a manifest-helm or manifest DeployItem has optional fields `deletionGroups` and
`deletionGroupsDuringUpdate`. They are lists whose items are objects with the fields:

- `predefinedResourceGroup`, optional, of type [predefined resource group](#predefined-resource-group).
- `customResourceGroup`, optional, of type [custom resource group](#custom-resource-group)

In each item, exactly one of the two fields must be set.

#### Predefined resource group

A predefined resource group is an object with the following fields:

- `type`, required, of type string. The supported values are:
  - `namespaced-resources`
  - `cluster-scoped-resources`
  - `crds`
  - `empty`

- `forceDelete`, optional, of type boolean, with default value `false`.

#### Custom resource group

A `customResourceGroup` is an object with the following fields:

- `resources`, required, a list as described in [resources of a custom resource group](#resources-of-a-custom-resource-group).
- `forceDelete`, optional, of type boolean, with default value `false`.
- `deleteAllResources`, optional, of type boolean, with default value `false`.

#### Resources of a custom resource group

The resources of a custom resource group are a list. Each item is an object with the following fields:

- `apiVersion`, required, of type string.
- `kind`, required, of type string.
- `names`, optional, a list of strings.
- `namespaces`, optional, a list of strings.
