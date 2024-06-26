apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Context
metadata:
  name: flux-installation
  namespace: ${namespace}

repositoryContext:
  baseUrl: ${repoBaseUrl}
  type: ociRegistry
