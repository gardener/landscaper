---
title: Component References
sidebar_position: 2
---

# Component References

In this example we describe how components could be reused in other components. 

This example is almost a copy of the [export-import example](../export-import). 
In that example we considered one component version with two resources, namely the blueprints of a root installation 
and its sub installations. In the present example, we consider two component versions, each with one blueprint resource.

The list of components in the [component-constructor.yaml](commands/component-constructor.yaml) has now two items:

- component `github.com/gardener/landscaper-examples/guided-tour/subinst-templates/root`, version `1.0.0`,
  with the blueprint of the root installation as resource.
- component `github.com/gardener/landscaper-examples/guided-tour/subinst-templates/sub`, version `1.0.0`,
  with the blueprint of the sub installation as resource.

Moreover, the first component version has a component reference to the second:

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

The three [sub installations](blueprints/root/subinstallation-1.yaml) specify their blueprint as follows:

```yaml
blueprint:
  ref: cd://componentReferences/sub/resources/blueprint-sub
```

The value of the field `blueprint.ref` has this structure:
- it starts with `cd://`, 
- followed by any number &ge; 0 of `/componentReferences/<name of component reference>`,
- and it ends with `/resources/<name of blueprint resource>`.

In our case we start in the root component, follow the component reference with name `sub`, 
and in the referenced component we select the resource with name `blueprint-sub`.
