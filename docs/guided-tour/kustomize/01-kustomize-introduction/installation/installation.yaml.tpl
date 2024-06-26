apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: flux
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:
  context: landscaper-examples

  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/guided-tour/kustomize/flux
      version: 1.0.0

  blueprint:
    ref:
      resourceName: blueprint

  imports:
    targets:
      - name: cluster
        target: my-cluster
