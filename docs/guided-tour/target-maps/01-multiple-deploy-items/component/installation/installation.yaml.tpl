apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: multiple-items
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:
  context: landscaper-examples

  componentDescriptor:
    ref:
      componentName: github.com/gardener/guided-tour/targetmaps/guided-tour-multiple-deploy-items
      version: 1.0.0

  blueprint:
    ref:
      resourceName: blueprint

  imports:
    targets:
      - name: clusters
        targetMap:
          red: cluster-red
          green: cluster-green
          blue: cluster-blue

    data:
      - name: config
        dataRef: config

  importDataMappings:
    namespace: ${targetNamespace}
