# Deployer Resource Health-/Readiness Checks

The Helm and Manifest deployers can check the readiness/health of the resources they just deployed. This document describes how the health checks work and how they can be configured.

Both types of health checks, default health checks and custom health checks, can only check the resources that have been deployed by the DeployItem they are a part of, i.e. those resources in the DeployItems `.status.providerStatus.managedResources` field.

**Index**:
- [Health Check Configuration](#health-check-configuration)
- [Default Health Checks](#default-health-checks)
- [Custom Health Checks](#custom-health-checks)

## Health Check configuration

To configure a health check for a DeployItem, the `.spec.healthChecks` field of the provider configuration can be set as follows.

```yaml
# Configuration of the health checks for the resources.
# optional
healthChecks:
  # Allows to disable the default health checks.
  # optional; set to false by default.
  disableDefault: true
  # Defines the time to wait before giving up on a resource
  # to be healthy. Should be changed with long startup time pods.
  # optional; default to 180 seconds/3 minutes.
  timeout: 3m
  # Configuration of custom health/readiness checks which are used
  # to check on custom fields and their values
  # especially useful for resources that came in through CRDs
  # optional
  custom:
  # the name of the custom health check, required
  - name: myCustomHealthcheck
    # timeout of the custom health check
    # optional, defaults to the timeout stated above
    timeout: 2m
    # temporarily disable this custom health check, useful for test setups
    # optional, defaults to false
    disabled: false
    # specific resources that should be selected for this health check to be performed on
    # a resource is uniquely defined by its GVK, namespace and name
    # required if no labelSelector is specified, can be combined with a labelSelector which is potentially harmful
    resourceSelector:
    - apiVersion: apps/v1
      kind: Deployment
      name: myDeployment
      namespace: myNamespace
    # multiple resources for the health check to be performed on can be selected through labels
    # they are identified by their GVK and a set of labels that all need to match
    # required if no resourceSelector is specified, can be combined with a resourceSelector which is potentially harmful
    labelSelector:
      apiVersion: apps/v1
      kind: Deployment
      matchLabels:
        app: myApp
        component: backendService
    # requirements specifies what condition must hold true for the given objects to pass the health check
    # multiple requirements can be given and they all need to successfully evaluate
    requirements:
    # jsonPath denotes the path of the field of the selected object to be checked and compared
    - jsonPath: .status.readyReplicas
      # operator specifies how the contents of the given field should be compared to the desired value
      # allowed operators are: DoesNotExist(!), Exists(exists), Equals(=, ==), NotEquals(!=), In(in), NotIn(notIn)
      operator: In
      # values is a list of values that the field at jsonPath must match to according to the operators
      values:
      - value: 1
      - value: 2
      - value: 3
```

## Default health checks

If the default health check is enabled for a DeployItem (which is the default), the following native Kubernetes resources will be checked as follows:

* `Pod`: It is considered healthy if it successfully completed
or if it has the the PodReady condition set to true.
* `Deployment`: It is considered healthy if the controller observed
its current revision and if the number of updated replicas is equal
to the number of replicas.
* `ReplicaSet`: It is considered healthy if its controller observed
its current revision and if the number of updated replicas is equal to the number of replicas.
* `StatefulSet`: It is considered healthy if its controller observed
its current revision, it is not in an update (i.e. UpdateRevision is empty)
and if its current replicas are equal to its desired replicas.
* `DaemonSet`: It is considered healthy if its controller observed
its current revision and if its desired number of scheduled pods is equal
to its updated number of scheduled pods.
* `ReplicationController`: It is considered healthy if its controller observed
its current revision and if the number of updated replicas is equal to the number of replicas.

## Custom Health Checks

Custom health checks can be used to match custom fields of selected resources to given values.

Multiple custom health checks can be defined for DeployItem as multiple resources to be checked each require a specifially tailored custom health check.

A custom health check must contain a name and a selector that selects the resource to be checked. This selector can either be:

- a set of `resourceSelector` each of which matches just one resource at a time, identified by its `apiVersion`, `kind`, `namespace` and `name`
- a `labelSelector` that matches multiple resources of the same `apiVersion` and `kind`, identified by a set of labels they need to have

As already mentioned above, these selectors can only select resources that have been deployed by the DeployItem the custom health check is a part of. They can only select resources that are listed in the DeployItems `.status.providerStatus.managedResources` field.

**WARNING:** It is possible and intended to provide both, `resourceSelector` and `labelSelector` so that resources with labels can be combined with other resources without labels in just one custom health check. However, it is left to the user to make sure that in this case, the condition to be checked can successfully evaluate for resources of different `apiVersion` and/or `kind`.

A field that is specified by its `jsonPath` will be extracted from each of the selected objects. The `jsonPath` is just a blank JSON path, i.e. without surrounding braces `{}`. The contents of the extracted field can be matched against a set of values according to an operator:

- `exists`: the given field must exist, irrelevant of its value
- `!` (NotExists): the given field must not exist
- `=` (Equals): the given field must match to the given value, only one value is allowed
- `!=` (NotEquals): the given fiels must _not_ match to the given value, only one value is allowed
- `in`: the given field must many to _at least one_ of the given values
- `notIn`: the given field mut _not_ match _to any_ of the given values

Allowed values are given as a list of key-value pairs with the key always being `value` and the value being a valid desired value. Values can be either primitives like ints, strings or bools as well as complex types.
