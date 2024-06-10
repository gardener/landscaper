---
title: Subinstallation Templates
sidebar_position: 4
---

# Subinstallation Templates

In this example we describe how sub installations could be created in a more dynamic way using templating, e.g.
depending on some input values. Therefore, we present a root installation with an import data mapping `numofsubinsts` 
containing an integer value, which creates as many sub installations as defined by the value of `numofsubinsts`. 
Every sub installation is quite simple and creates one deploy item which deploys one config map.

The list of components in the [component-constructor.yaml](commands/component-constructor.yaml) has two items:

- component `github.com/gardener/landscaper-examples/guided-tour/subinst-templates/root`, version `1.0.0`,
  with the blueprint of the root installation as resource.
- component `github.com/gardener/landscaper-examples/guided-tour/subinst-templates/sub`, version `1.0.0`,
  with the blueprint of the sub installation as resource.

The first component version has a component reference to the second:

```yaml
componentReferences:
  - name: sub
    componentName: github.com/gardener/landscaper-examples/guided-tour/subinst-templates/sub
    version: 1.0.0
```

The [root installation](installation/installation.yaml.tpl) specifies its blueprint in the usual way:

```yaml
spec:
  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/guided-tour/subinst-templates/root
      version: 1.0.0
  blueprint:
    ref:
      resourceName: blueprint-root
```

The root installation defines the number of sub installations which should be created in its import data mappings:

```yaml
  importDataMappings:
    numofsubinsts: 3
```

The [blueprint of the root installation](blueprints/root/blueprint.yaml) has `numofsubinsts` as input parameter
and a special section which creates the sub installations via templating:

```yaml
subinstallationExecutions:
  - name: subinst-executions
    type: GoTemplate
    file: /subinst-execution.yaml
```

The specific template specifications for the sub installations is stored in a [file](blueprints/root/subinst-execution.yaml)
which looks as follows and creates `numofsubinsts` times a sub installation with name `subinst-<loop-number>`. 

```yaml
subinstallations:
{{- range $index := .imports.numofsubinsts | int | until }}
  - apiVersion: landscaper.gardener.cloud/v1alpha1
    kind: InstallationTemplate
    name: subinst-{{ $index }}
    blueprint:
      ref: cd://componentReferences/sub/resources/blueprint-sub

    imports:
      targets:
        - name: cluster
          target: cluster

    importDataMappings:
      configmap-name-in: cm-{{ $index }}
{{ end }}
```

Every sub installation gets its own import value in the `importDataMappings` section which is used as the config map name
by the corresponding deploy items.

The blueprint for the sub installations could be found  [here](blueprints/sub/blueprint.yaml).

## Procedure

1. On the target cluster, create a namespace `example`. It is the namespace into which we will deploy the ConfigMaps.

2. On the Landscaper resource cluster, create a namespace `cu-example`.

3. On the Landscaper resource cluster, in namespace `cu-example`, create a Target `my-cluster` containing a
   kubeconfig for the target cluster, a Context `landscaper-examples`, 
   and an Installation `subinst-templates`. There are templates for these resources in the directory
   [installation](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/subinstallations/subinst-templates/installation).
   To apply them:
  - adapt the [settings](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/subinstallations/subinst-templates/commands/settings) file
    such that the entry `RESOURCE_CLUSTER_KUBECONFIG_PATH` points to the kubeconfig of the resource cluster,
    and the entry `TARGET_CLUSTER_KUBECONFIG_PATH` points to the kubeconfig of the target cluster,
  - run the [deploy-k8s-resources script](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/subinstallations/subinst-templates/commands/deploy-k8s-resources.sh),
    which will template and apply the Target, Context, and Installation.

As a result, the three sub installations have deployed three ConfigMaps on the target cluster in namespace `example`:

```shell
$ kubectl get configmaps -n example

NAME                    DATA   AGE
cm-0               1      53s
cm-1               1      53s
cm-2               1      53s
```

## Cleanup

You can remove the Installation with the
[delete-installation script](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/subinstallations/subinst-templates/commands/delete-installation.sh).
When the Installation is gone, you can delete the Context and Target with the
[delete-other-k8s-resources script](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/subinstallations/subinst-templates/commands/delete-other-k8s-resources.sh).

