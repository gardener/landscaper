apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: flux-podinfo
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:
  context: flux-podinfo

  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/guided-tour/flux-podinfo
      version: 1.0.0

  blueprint:
    ref:
      resourceName: blueprint

  imports:
    targets:
      - name: cluster
        target: my-cluster
