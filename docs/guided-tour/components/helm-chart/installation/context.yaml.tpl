apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Context
metadata:
  name: ${contextName}
  namespace: ${namespace}

repositoryContext:
  baseUrl: ${repoBaseUrl}
  type: ociRegistry
