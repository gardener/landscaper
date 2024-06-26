apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: flux-installation
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:
  context: flux-installation

  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/guided-tour/flux-installation
      version: 1.0.0

  blueprint:
    ref:
      resourceName: blueprint

  imports:
    targets:
      - name: cluster
        target: my-cluster
