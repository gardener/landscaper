# Deploy Targets

## Motivation

Possible installations include components that may run in fenced environments that cannot be accessed by the landscaper cluster.
Therefore, DeployItem need to specify a deployer that reside in the fenced environment in order to deploy the component.

## DeployItems

DeployItems are the connection between the landscaper and the deployers which means that the targeted environment has to 
be specified in the DeployItem.

DeployItems are automatically created via the executors that are defined in the ComponentDefinition.

As it is possible for Components to target multiple environments, it is required that the deploy items of a Component can target different environments.
Each environment has a landscapelet(harvester) unique name that is also propagated to all of its managed deployers.
The deployers have to implement the functionaility to only reconcile DeployItem macthcing their "deployer class".

The contract then is as follows.<br>
The environment target can be optionally defined for a step in the execution.
If non is given, the default deployers with no "deployer class" will reconcile the item.

Additional Landscapelets with their unique names have to be manually configured by the operator.
Therefore, the targets have to be specified in the external configuration.

*Example Seed/Topology*

Definition:
```yaml
kind: Definition
name: topology
export:
- key: seeds
  type: seedArray
```

External Config:
```yaml
seeds:
- name: my fenced seed
  target: my-fenced-env
```

```yaml
kind: ComponentDefinition
executors: |
- type: Helm
  config: test
```

seeds:
```yaml
kind: ComponentDefinition
executors: |
{{ range topology.seeds }}
- type: Container
  target: {{ .target }}
  config: 
    seedconfig: .config
{{ end }}
```

- Annotation in the deploy item
```yaml
kind: DeployItem
metadata:
    annotations:
      deployer.landscaper.gardener.cloud/class: my-fenced-env
spec:
   config:
     kubeconfig: apiVer... # {{ .imports.kubeconfig }}
```



### Other ideas

#### own spec with special target

```yaml
soils:
- name: my fenced soil
  targets: 
    type: kubernetes-cluster
    name: fenced-env-1
    config: 
      kubeconfig: apiVer....
shootedseed:
- name: my fenced seed
  target: 
    type: kubernetes-cluster
    name: gardener # create new target
```
```yaml
kind: DeployItem

spec:
  type: Helm

  target: # {{ .imports.target }}
    type: kubernetes-cluster
    name: fenced-env-1
    config:
      my-special-config: abc
      kubeconfig: apiVers...
```

### target as its own resource

```yaml
kind: Blueprint

import:
- key: cluster1
  type: targetRef
  target: fenced-env-1 # => where to define target types
- key: cluster1
  type: kubeconfig
- key: etcd # optional as own property
  component: 
    ref: github.com/gardener/mcm
    kind: externalResource
    resource: etcd

exports:
- key: cluster1
  type: targetName # => get target <targetName>
- key: cluster3
  target: fenced-env-1 # => where to define target types
- key: cluster4
  target: k8s # => where to define target types
```


reusable targets
```yaml
kind: Target
metadata:
  name: fenced-env-1
  annotations:
    "my-an": "abc"
spec:
  type: kubernetes-cluster
  subtypes: 
  - type: gardener
    providerConfig:
      topology: 
        seeds:
        - abc
  - type: apphub
      
  config:
    my-special-config: abc
    kubeconfig: apiVers...
```

```yaml
type: aws
providerConfig: any
```

```yaml
kind: DeployItem
spec:
  type: Helm
  targetRef: fenced-env-1
```

- extra mapping for deployer to configure responsible targets (also possible during runtime)
- or static commandline option
```yaml
kind: TargetMapping
metadata:
    labels:
      resp.class: "abc"
spec:
- target: name
- target: fenced-env-1
```