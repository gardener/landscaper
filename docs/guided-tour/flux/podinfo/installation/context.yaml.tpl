apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Context
metadata:
  name: flux-podinfo
  namespace: ${namespace}

repositoryContext:
  baseUrl: ${repoBaseUrl}
  type: ociRegistry
