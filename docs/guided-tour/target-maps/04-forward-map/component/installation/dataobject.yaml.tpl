apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DataObject
metadata:
  name: config
  namespace: ${namespace}
data:
  blue:
    color: blue
    cpu: 100m
    memory: 100Mi

  green:
    color: green
    cpu: 120m
    memory: 120Mi

  yellow:
    color: yellow
    cpu: 140m
    memory: 140Mi

  orange:
    color: orange
    cpu: 160m
    memory: 160Mi

  red:
    color: red
    cpu: 180m
    memory: 180Mi

