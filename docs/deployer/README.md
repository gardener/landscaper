# Deployer

The landscaper offloads most work to so called Deployers.
These deployers are meant to have specific deploy, install and update logic.

## List of Deployers

The following deployers are available out of the box with the Landscaper. Further deployer could be implemented and registered.

- [Mock](mock.md)
- [Helm](helm.md)
- [Kubernetes Manifest](manifest.md)
- [Container](container.md)


## Common Documentation

### Target Selector

If multiple instances of a deployer are used (e.g. when dealing with different environment) by default all instances will concurrently reconcile the deploy items.
Therefore, a mechanism is needed that defines the responsibility for the different deploy items and deployer instances.

There are different possible options for how this can be achieved, which could also be very deployer/target specific (e.g. the url or aws region).
The Landscaper deployer library currently offers some standardized default options by using targets to determine responsibility.

**Annotations/Labels**

A simple mechanism is to annotate the targets (either using annotations or labels) and configure each deployer instance 
with a different target selector that matches these annotations (or labels). This is comparable to how Kubernetes Ingress 
objects are selected.

The landscaper offers a default implementation to use label selector to either select targets based on annotations or labels. 
(See the official Kubernetes documentation for detailed documentation on how the selectors work: 
https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors)

```yaml
--- # target
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  annotations:
    "my-deployer-env": "internal"
spec: ...
--- # selector configuration
selector:
  annotations:
  - key: "my-deployer-env"
    operation: "="
    values:
    - "internal"
```
