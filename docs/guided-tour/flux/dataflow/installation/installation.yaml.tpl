apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: flux-dataflow
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:
  context: flux-dataflow

  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/guided-tour/flux-dataflow
      version: 1.0.0

  blueprint:
    ref:
      resourceName: blueprint-first

  imports:
    targets:
      - name: cluster
        target: my-cluster

  exports:
    data:
      - name: token
        dataRef: my-token
