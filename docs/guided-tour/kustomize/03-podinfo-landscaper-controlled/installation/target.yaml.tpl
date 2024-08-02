apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  name: ${name}
  namespace: ${namespace}
spec:
  type: landscaper.gardener.cloud/kubernetes-cluster
  config:
    kubeconfig: |
${kubeconfig}