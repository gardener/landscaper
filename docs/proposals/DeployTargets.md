# Deploy Targets

## Motivation

Possible installations include components that may run in fenced environments that cannot be accessed by the landscaper cluster.
Therefore, DeployItems need to specify a deployer that reside in the fenced environment in order to deploy the component.

### Other ideas

#### own spec with special target

```yaml
kind: DeployItem

spec:
  type: Helm

  target: <name of the target> # maybe also whole target
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
  targetType: landscaper.cloud/kubernetes-cluster # => get target <targetName>
- key: cluster3
  targetType: landscaper.cloud/kubernetes-cluster # => where to define target types
- key: cluster4
  target: landscaper.cloud/kubernetes-cluster # => where to define target types
```


reusable targets
```yaml
kind: Target
metadata:
  name: fenced-env-1
  annotations:
    "my-an": "abc"
spec:
  type: landscaper.cloud/kubernetes-cluster
  subtypes: 
  - type: gardener
    providerConfig:
      topology: 
        seeds:
        - abc
  - type: apphub
  properties:
    myprop: ...
  config:
    my-special-config: abc
    kubeconfig: apiVers...
```

Not needed now, need to be evaluated when we have more experience.
```yaml
kind: TargetType
metadata:
  name: landscaper.cloud/kubernetes-cluster
spec:
  apiType: landscaper.cloud/kubernetes-cluster
  jsonschema:
    type: object
    properties:
      kubeconfig:
```

Deployers can be matched to different targets with differnet mechanisms.
We as landscaper should provide a library where multiple possibilites are built in to mapp target -> deployer.
Possibilities:
- Owner objects (see external dns provider)
- Class Annotation
- 1 to 1 mapping for target name -> deployer

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