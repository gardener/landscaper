apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: export-token
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:
  context: landscaper-examples

  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/guided-tour/export-token
      version: 1.1.0

  blueprint:
    ref:
      resourceName: blueprint

  # Set values for the import parameters of the blueprint
  imports:
    targets:
      - name: cluster        # name of an import parameter of the blueprint
        target: my-cluster   # name of the Target custom resource containing the kubeconfig of the target cluster

  exports:
    data:
      - name: token
        dataRef: my-token
