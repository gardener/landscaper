apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: flux-dataflow-repo
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
      resourceName: blueprint-repo

  imports:
    targets:
      - name: cluster
        target: my-cluster

  exports:
    data:
      - name: gitRepositoryName
        dataRef: git-repository-name
