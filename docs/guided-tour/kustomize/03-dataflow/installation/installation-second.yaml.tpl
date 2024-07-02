apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: dataflow-second
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
      resourceName: blueprint-second

  imports:
    targets:
      - name: cluster
        target: my-cluster
    data:
      - name: gitRepositoryName
        dataRef: git-repository-name
      - name: token
        dataRef: my-token
