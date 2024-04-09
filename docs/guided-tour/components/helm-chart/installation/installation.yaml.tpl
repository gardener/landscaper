apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: echo-server
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:
  context: landscaper-examples

  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/guided-tour/helm-chart
      version: 1.0.0

  blueprint:
    ref:
      resourceName: blueprint

  # Set values for the import parameters of the blueprint
  imports:
    targets:
      - name: cluster           # name of an import parameter of the blueprint
        target: my-cluster      # name of the Target custom resource containing the kubeconfig of the target cluster
