components:
  - name: github.com/gardener/landscaper-examples/guided-tour/protected-helm-repo
    version: 1.0.0
    provider:
      name: internal
    resources:
      - name: blueprint
        type: landscaper.gardener.cloud/blueprint
        input:
          type: dir
          path: ../blueprint
          compress: true
          mediaType: application/vnd.gardener.landscaper.blueprint.v1+tar+gzip
      - name: chart
        type: helmChart
        version: 1.0.0
        access:
          type: helm
          helmChart: ${helmChart}
          helmRepository: ${helmRepository}
