---
title: TargetList Imports
sidebar_position: 8
---

# TargetList Imports

While the Landscaper allows to import lists of Targets into Installations/Blueprints, it is not intuitively clear how they can be used. This documentation will provide a few examples for this. Please note that the provided example YAMLs often are redacted to show only the part of the spec which is of interest for demonstrating the usage of TargetList imports.

**Index**:
- [TargetList Imports](#targetlist-imports)
  - [The Basics](#the-basics)
    - [TargetList Import Declarations in Blueprints](#targetlist-import-declarations-in-blueprints)
    - [TargetList Imports in Installations](#targetlist-imports-in-installations)
    - [Referencing (Targets from) TargetList Imports](#referencing-targets-from-targetlist-imports)
      - [In DeployItems](#in-deployitems)
      - [In Nested Installations](#in-nested-installations)
        - [Whole TargetList](#whole-targetlist)
      - [Single Target from TargetList](#single-target-from-targetlist)
  - [Usage Examples](#usage-examples)
    - [One DeployItem per Target](#one-deployitem-per-target)
      - [GoTemplate](#gotemplate)
      - [Spiff](#spiff)
    - [One Nested Installation per Target](#one-nested-installation-per-target)
      - [GoTemplate](#gotemplate-1)
      - [Spiff](#spiff-1)


## The Basics

### TargetList Import Declarations in Blueprints

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

imports:
- name: mytargets
  type: targetList
  targetType: landscaper.gardener.cloud/test
```

A TargetList import can be declared in a Blueprint simply by using `targetList` as type for the import.


### TargetList Imports in Installations

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: targetlist
spec:
  imports:
    targets:
    - name: mytargets
      targets:
      - target1
      - target2
      - target3
```

To satisfy a Blueprint's `targetList` import in an Installation, simply provide a list of the Targets' names. All targets have to exist in the same namespace as the installation and must have the same `targetType` - the one specified in the Blueprint.

> While probably rarely useful, it is possible to list the same Target multiple times.


### Referencing (Targets from) TargetList Imports

#### In DeployItems

To reference a Target from a TargetList in a DeployItem's `target` field, use the name of the TargetList import and its index.
```yaml
imports:
- name: mytargets
  type: targetList
  targetType: landscaper.gardener.cloud/kubernetes-cluster

deployExecutions:
- name: deploy-executions
  type: Spiff
  template:
    deployItems:
    - name: my-first-target-di
      type: landscaper.gardener.cloud/kubernetes-manifest
      target:
        import: mytargets
        index: 0
```

#### In Nested Installations

##### Whole TargetList

The special field `targetListRef` can be used to forward the complete TargetList import of a parent Installation to its subinstallation.
```yaml
imports:
- name: mytargets
  type: targetList
  targetType: landscaper.gardener.cloud/kubernetes-cluster

subinstallations:
- apiVersion: landscaper.gardener.cloud/v1alpha1
  kind: InstallationTemplate
  name: mysubinst
  imports:
    targets:
    - name: also-my-targets # assuming the nested installation's blueprint expects a TargetList import named 'also-my-targets'
      targetListRef: mytargets
```

#### Single Target from TargetList

To use a single Target from a parent Installation's TargetList import, reference it by using the name of the TargetList import, followed by the index in square brackets.
```yaml
imports:
- name: mytargets
  type: targetList
  targetType: landscaper.gardener.cloud/kubernetes-cluster

subinstallations:
- apiVersion: landscaper.gardener.cloud/v1alpha1
  kind: InstallationTemplate
  name: mysubinst
  imports:
    targets:
    - name: my-single-target # assuming the nested installation's blueprint expects a single target import named 'my-single-target'
      target: "mytargets[0]"
```

This notation can also be used to compose new TargetLists:
```yaml
imports:
- name: myfirsttargetlist
  type: targetList
  targetType: landscaper.gardener.cloud/kubernetes-cluster
- name: mysecondtargetlist
  type: targetList
  targetType: landscaper.gardener.cloud/kubernetes-cluster

subinstallations:
- apiVersion: landscaper.gardener.cloud/v1alpha1
  kind: InstallationTemplate
  name: mysubinst
  imports:
    targets:
    - name: also-my-targets # assuming the nested installation's blueprint expects a TargetList import named 'also-my-targets'
      targets:
      - "myfirsttargetlist[0]"
      - "mysecondtargetlist[4]"
```


## Usage Examples

See also the [landscaper examples](https://github.com/gardener/landscaper-examples/tree/master/targetlists) for full example Installations.

### One DeployItem per Target

This is probably the most prominent use-case of the feature: The Installation is supposed to perform the same action for each Target of an imported TargetList. For example, the TargetList contains kubernetes-cluster targets - basically kubeconfigs - and the Installation should deploy a manifest into each of the referenced clusters.

To achieve this, use one of the offered templating options to render multiple DeployItems. Both of the examples specified below will result in one DeployItem per entry in the imported TargetList.

#### GoTemplate
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

imports:
- name: mytargets
  type: targetList
  targetType: landscaper.gardener.cloud/kubernetes-cluster

deployExecutions:
- name: deploy-executions
  type: GoTemplate
  template: |
    deployItems:
    {{ range $idx, $target := .imports.mytargets }}
    - name: my-di-{{ $idx }}-{{ $target.metadata.name }}
      type: landscaper.gardener.cloud/kubernetes-manifest
      target:
        import: mytargets
        index: {{ $idx }}
      config:
        apiVersion: manifest.deployer.landscaper.gardener.cloud/v1alpha2
        kind: ProviderConfiguration
        manifests:
        - policy: manage
          manifest:
            apiVersion: v1
            kind: Secret
            metadata:
              name: foo-{{ $idx }}
              namespace: default
            data:
              foo: YmFy
    {{ end }}
```

#### Spiff
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

imports:
- name: mytargets
  type: targetList
  targetType: landscaper.gardener.cloud/kubernetes-cluster

deployExecutions:
- name: deploy-executions
  type: Spiff
  template:
    deployItems: (( sum[imports.mytargets|[]|s,idx,tar|-> s *diTemplate] ))
    diTemplate:
      <<<: (( &template &temporary ))
      name: (( "my-di-" idx "-" tar.metadata.name ))
      type: landscaper.gardener.cloud/kubernetes-manifest
      target:
        import: mytargets
        index: (( idx ))
      config:
        apiVersion: manifest.deployer.landscaper.gardener.cloud/v1alpha2
        kind: ProviderConfiguration
        manifests:
        - policy: manage
          manifest:
            apiVersion: v1
            kind: Secret
            metadata:
              name: (( "foo-" idx ))
              namespace: default
            data:
              foo: YmFy
```
Note that it is not possible to name the variable which holds the current Target (`tar` in this case) `target`, as _Spiff_ will confuse this with the node named `target` in the DeployItem template (`diTemplate`).


### One Nested Installation per Target

It is also possible to create one nested installation per Target in a TargetList.

#### GoTemplate
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

imports:
- name: mytargets
  type: targetList
  targetType: landscaper.gardener.cloud/kubernetes-cluster

subinstallationExecutions:
- name: subinst-executions
  type: GoTemplate
  template: |
    subinstallations:
    {{ range $idx, $target := .imports.mytargets }}
    - apiVersion: landscaper.gardener.cloud/v1alpha1
      kind: InstallationTemplate
      name: my-subinst-{{ $idx }}-{{ $target.metadata.name }}
      imports:
        targets:
        - target: "mytargets[{{ $idx }}]"
          name: mytarget
    {{ end }}
```

#### Spiff
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

imports:
- name: mytargets
  type: targetList
  targetType: landscaper.gardener.cloud/kubernetes-cluster

subinstallationExecutions:
- name: subinst-executions
  type: Spiff
  template:
    subinstallations: (( sum[imports.mytargets|[]|s,idx,tar|-> s *iTemplate] ))
    iTemplate:
      <<<: (( &template &temporary ))
      apiVersion: landscaper.gardener.cloud/v1alpha1
      kind: InstallationTemplate
      name: (( "my-subinst-" idx "-" tar.metadata.name ))
      imports:
        targets:
        - target: (( "mytargets[" idx "]" ))
          name: mytarget
```
