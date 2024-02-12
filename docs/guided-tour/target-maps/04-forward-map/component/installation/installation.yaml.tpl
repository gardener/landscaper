apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: forward-map
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:
  context: landscaper-examples

  componentDescriptor:
    ref:
      componentName: github.com/gardener/guided-tour/targetmaps/guided-tour-forward-map
      version: 1.0.0

  blueprint:
    ref:
      resourceName: blueprint-root

  imports:
    targets:
      - name: redRootCluster
        target: cluster-red
      - name: blueRootCluster
        target: cluster-blue
      - name: greenRootCluster
        target: cluster-green

  importDataMappings:
    namespace: ${targetNamespace}
