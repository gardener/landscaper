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

Currently, there is no predefined order in which the manifests are removed. The removal of a Chart is successful if 
all objects were gone.

We propose the following solution to have more control over the deletion process:

- The basic order of the manifest removal is:
  - Custom resources (CRs) of CRDs deployed by the Chart (namespaced and not namespaced CRs) (deletionRank=100)
  - CRs of CRDs not deployed by the Chart (namespaced and not namespaced CRs) (deletionRank=200)
  - Namespaced objects (deletionRank=300)
  - Not namespaced objects (except CRDs) (deletionRank=400)
  - CRDs (deletionRank=500)
  - Namespaces (deletionRank=600)

The deletion continues only with the next object group if all objects from the groups before are gone.

Each of these standard groups of objects have a particular deletionRank which determines the deletion order. To change 
the deletion order of particular resources you could overwrite their default deletionRank in the specification of a 
DeployItem as shown next:

```
deployItem:
...
  - deletionOrderRules:
    - deletionRank: <some integer number>
      types:
        - <group/version/kind of CR X>
```

By such a deletion order rule, the objects specified get the new deletionRank of the rule, and are deleted after all 
other objects with a lower deletionRank but before those with a higher deletionRank. Thereby the deletion of objects 
with higher rankings is only continued if all objects with a deletionRank lower or equal to the deletionRank, specified 
in the rule, are gone.


Another important point is the possibility to add some wait periods or forced cleanup after the deletion of a particular 
deletionRank, e.g. to give an operator some time to clean up time. Such rules will have the following format:

```
deployItem:
...
  - deletionOrderRules:
    - deletionRank: <some integer number>
      types:
        - <group/version/kind of CR X>
      waitTimeout: <optional, defaults to 1 minute>
      forceDeleteAfterWaitTimeout: # optional
        enabled: <true/false>
```

The meaning of the additional fields is the following:

- waitTimeout: Duration the deployer waits before it continues, either with the deletion of objects with a higher 
  deletionRank or if specified with the forceDeleteAfterWaitTimeout step of the rule. 

- forceRemoveAfterWaitPeriod: If enabled, the finalizers of all remaining objects with a deletionRank lower or equal  
  the one of the rule, are removed.

Of course, it is also possible to specify wait periods or forced cleanup for the standard groups, e.g. the following
example introduces such a rule for the CRs of CRDs specified in the Chart, which have the default deletionRank of 100. 
You see that the types section is skipped in this case.

```
deployItem:
...
  - deletionOrderRules:
    - deletionRank: 100
      waitTimeout: <optional, defaults to 1 minute>
      forceDeleteAfterWaitTimeout: # optional
        forceTimeout: <some duration>
```


Open questions: 
- How to handle different CRs versions of one CRD?
- How to handle deletions in Chart upgrades?