apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: subinst-templates
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:
  context: landscaper-examples

  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/guided-tour/subinst-templates/root
      version: 1.0.0

  blueprint:
    ref:
      resourceName: blueprint-root

  imports:
    targets:
      - name: cluster
        target: my-cluster

  importDataMappings:
    numofsubinsts: 3

