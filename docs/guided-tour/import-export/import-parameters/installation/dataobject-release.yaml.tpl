apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DataObject
metadata:
  name: my-release
  namespace: ${namespace}
data:
  name: echo
  namespace: example
