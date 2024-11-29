apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: self-inst
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:

  imports:
    targets:
      - name: cluster
        target: self-target

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
                    type: landscaper.gardener.cloud/kubernetes-manifest
          
                    target:
                      import: cluster
          
                    config:
                      apiVersion: manifest.deployer.landscaper.gardener.cloud/v1alpha2
                      kind: ProviderConfiguration
                      updateStrategy: update
                      manifests:
                        - policy: manage
                          manifest:
                            apiVersion: v1
                            kind: ConfigMap
                            metadata:
                              name: self-target-example
                              namespace: example
                            data:
                              testData: hello
