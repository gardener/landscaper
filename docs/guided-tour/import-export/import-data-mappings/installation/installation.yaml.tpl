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
      componentName: github.com/gardener/landscaper-examples/guided-tour/echo-server
      version: 2.2.0

  blueprint:
    ref:
      resourceName: blueprint

  # Set values for the import parameters of the blueprint
  imports:
    targets:
      - name: cluster        # name of an import parameter of the blueprint
        target: my-cluster   # name of the Target custom resource containing the kubeconfig of the target cluster

    data:
      - name: temp-name                 # name used in the import data mapping
        dataRef: my-release-name        # name of a DataObject containing the parameter value

      - name: temp-namespace            # name used in the import data mapping
        dataRef: my-release-namespace   # name of a DataObject containing the parameter value

  importDataMappings:
    release:
      name: (( temp-name ))
      namespace: (( temp-namespace ))
    values:
      text: hello
