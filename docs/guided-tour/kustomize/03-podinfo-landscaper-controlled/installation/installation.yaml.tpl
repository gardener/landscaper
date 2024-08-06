apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: podinfo
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:
  context: landscaper-examples

  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/guided-tour/kustomize/podinfo-landscaper-controlled
      version: 1.0.0

  blueprint:
    ref:
      resourceName: blueprint

  imports:
    targets:
      - name: cluster
        target: my-cluster
      - name: cluster2
        target: my-cluster-2
