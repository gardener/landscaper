apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  name: self-target
  namespace: ${namespace}
spec:
  type: landscaper.gardener.cloud/kubernetes-cluster
  config:
    selfConfig:
      serviceAccount:
        name: self-serviceaccount
      expirationSeconds: 3600
