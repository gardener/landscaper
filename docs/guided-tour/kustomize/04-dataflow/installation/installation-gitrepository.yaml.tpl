apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: dataflow-gitrepository
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:
  context: landscaper-examples

  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/guided-tour/kustomize/dataflow
      version: 1.0.0

  blueprint:
    ref:
      resourceName: blueprint-gitrepository

  imports:
    targets:
      - name: cluster
        target: my-cluster

  exports:
    data:
      - name: gitRepositoryName
        dataRef: gitrepository-name
