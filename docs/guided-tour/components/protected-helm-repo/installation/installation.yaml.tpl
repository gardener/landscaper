apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: protected-helm-repo
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:
  context: landscaper-examples-protected-helm-repo

  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/guided-tour/protected-helm-repo
      version: 1.0.0

  blueprint:
    ref:
      resourceName: blueprint

  imports:
    targets:
      - name: cluster
        target: my-cluster
