apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Context
metadata:
  name: landscaper-examples
  namespace: ${namespace}

repositoryContext:
  baseUrl: ${repoBaseUrl}
  type: ociRegistry
