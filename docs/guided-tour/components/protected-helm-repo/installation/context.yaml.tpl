apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Context
metadata:
  name: landscaper-examples-protected-helm-repo
  namespace: ${namespace}

repositoryContext:
  baseUrl: ${repoBaseUrl}
  type: ociRegistry

configurations:
  helmChartRepoCredentials:
    auths:
      - url: ${helmUrlPrefix}
        authHeader: ${helmAuthHeader}
