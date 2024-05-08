apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: export-import
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:
  context: landscaper-examples

  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/guided-tour/export-import
      version: 1.0.0

  blueprint:
    ref:
      resourceName: blueprint-root

  imports:
    targets:
      - name: cluster
        target: my-cluster

    data:
      - name: configmap-name-base
        dataRef: do-configmap-name-base

  exports:
    data:
      - name: configmap-names
        dataRef: do-configmap-names
