apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: multiple-subinst
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:
  context: landscaper-examples

  componentDescriptor:
    ref:
      componentName: github.com/gardener/guided-tour/targetmaps/guided-tour-multiple-subinst
      version: 1.0.0

  blueprint:
    ref:
      resourceName: blueprint-root

  imports:
    targets:
      - name: rootclusters
        targetMap:
          red: cluster-red
          green: cluster-green
          blue: cluster-blue
    data:
      - name: rootconfig
        dataRef: config

  importDataMappings:
    namespace: ${targetNamespace}
