apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: hello-world
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:

  # Set values for the import parameters of the blueprint
  imports:
    targets:
      - name: cluster
        target: my-cluster

  blueprint:
    inline:
      filesystem:
        blueprint.yaml: |
          apiVersion: landscaper.gardener.cloud/v1alpha1
          kind: Blueprint
          jsonSchema: "https://json-schema.org/draft/2019-09/schema"

          imports:
            - name: cluster
              type: target
              targetType: landscaper.gardener.cloud/kubernetes-cluster

          deployExecutions:
            - name: default
              type: GoTemplate
              template: |
                deployItems:
                  - name: default-deploy-item
                    type: landscaper.gardener.cloud/helm
          
                    target:
                      import: cluster
          
                    timeout: 2m   # progressing timeout of the DeployItem
          
                    config:
                      apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
                      kind: ProviderConfiguration
                      name: hello-world
                      namespace: example
                      createNamespace: true
                      chart:
                        ref: eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:1.0.0  # version exists
                      values:
                        testData: hello
