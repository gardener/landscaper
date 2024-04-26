apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: hello-world
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:

  # Set values for the two import parameters of the blueprint
  imports:
    targets:
      - name: cluster-1
        target: my-cluster-1
      - name: cluster-2
        target: my-cluster-2

  blueprint:
    inline:
      filesystem:
        blueprint.yaml: |
          apiVersion: landscaper.gardener.cloud/v1alpha1
          kind: Blueprint
          jsonSchema: "https://json-schema.org/draft/2019-09/schema"

          # Define two import parameters
          imports:
            - name: cluster-1
              type: target
              targetType: landscaper.gardener.cloud/kubernetes-cluster
            - name: cluster-2
              type: target
              targetType: landscaper.gardener.cloud/kubernetes-cluster

          deployExecutions:
            - name: default
              type: GoTemplate
              template: |
                deployItems:
                  - name: deploy-item-1
                    type: landscaper.gardener.cloud/helm

                    target:
                      import: cluster-1   # Use first import parameter

                    config:
                      apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
                      kind: ProviderConfiguration
                      name: hello-world-1
                      namespace: example-1
                      createNamespace: true
                      chart:
                        ref: eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:1.0.0
                      values:
                        testData: foo

                  - name: deploy-item-2
                    type: landscaper.gardener.cloud/helm
                    
                    dependsOn:
                      - deploy-item-1

                    target:
                      import: cluster-2  # Use second import parameter

                    config:
                      apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
                      kind: ProviderConfiguration
                      name: hello-world-2
                      namespace: example-2
                      createNamespace: true
                      chart:
                        ref: eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:1.0.0
                      values:
                        testData: bar
