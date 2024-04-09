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
      - name: cluster        # name of an import parameter of the blueprint
        target: my-cluster   # name of the Target custom resource containing the kubeconfig of the target cluster

  blueprint:
    inline:
      filesystem:
        blueprint.yaml: |
          apiVersion: landscaper.gardener.cloud/v1alpha1
          kind: Blueprint
          jsonSchema: "https://json-schema.org/draft/2019-09/schema"

          imports:
            - name: cluster   # name of the import parameter
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
                      import: cluster   # "cluster" is the name of an import parameter
          
                    config:
                      apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
                      kind: ProviderConfiguration
                      name: hello-world
                      namespace: example
                      createNamespace: true
                      chart:
                        ref: eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:1.0.1
                      values:
                        testData: hello
