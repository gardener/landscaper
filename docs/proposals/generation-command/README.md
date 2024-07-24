# Generation Command for Helm Chart Use-Cases 

We propose new Landscaper CLI commands to generate the resources for simple Landscaper scenarios.
Supported scenarios are the deployment of one or more Helm charts. Moreover, we support a data flow between these charts.


## Examples

Example [01-single-helm-chart](./01-single-helm) discusses the deployment of a single helm chart using the
proposed commands.

Example [02-double-helm](./02-double-helm) discusses the deployment of two helm charts. 
The first of them exports values which are then imported by the second.

Example [03-units](./03-units) adds "external" imports and exports, so that there is not only a dataflow between the
charts, but moreover data can also be imported from outside, resp. exported. 


## Command to generate blueprints and component constructor

```shell
landscaper-cli blueprint create [CONFIG_FILE_PATH] [RESULT_DIRECTORY_PATH]
```

The command uses a [config file](#the-config-file) which contains all necessary data for the generation. 
The command generates several blueprints into the given result directory: one for each Helm deployment, plus a root 
blueprint. In addition, it generates a component constructor, that will contain the blueprints, charts, and images as
resources. Thus, the result directory will look like this:

```shell
[result directory]
├── blueprints
│   ├── blueprint-root
│   │   └── blueprint.yaml
│   ├── blueprint-my-first-item
│   │   └── blueprint.yaml
│   ├── blueprint-my-second-item
│   │   └── blueprint.yaml
│   ...
└── component-constructor.yaml
```

### The config file

The config file specifies:
- the OCM component name and version.
- the base URL of the OCM repository (just needed to generate an upload command).
- a map of item names to item definitions. Each item defines the deployment of a Helm chart.

```yaml
component:
  repositoryBaseUrl: eu.gcr.io/gardener-project/landscaper/examples
  name: github.com/gardener/landscaper-examples/guided-tour/automation/simple-helm
  version: 1.0.0

items:
  my-first-item:   # item name
    ...            # item definition
  
  my-second-item:
    ...
  
  ...
```

An item definition specifies the chart and the image(s):

```yaml
items:
  my-first-item:
    type: helm
    createNamespace: true
    chart:
      name: echo-server-chart
      access:
        type: ociArtifact
        imageReference: eu.gcr.io/gardener-project/landscaper/examples/charts/guided-tour/echo-server-extended:1.0.0
    images:
      echo-server-image: hashicorp/http-echo:0.2.3
    additionalValues: |
      path:
        to:
          image: {{ $images.echo-server-image }}
      foo: bar
```

Chart and images will be added to the component constructor. 

Helm values are usually provided by the Installation. However, an item definition has a field `additionalValues`, where 
one can specify Helm values which are Installation independent. They are merged with the values from the Installation. 

Images can be set in the `additionalValues`. 
The value of field `additionalValues` is a GoTemplate. In this template one can use a predefined variable `$images`, 
which contains the `images` map of the item definition.

There is an [example of a config.yaml](../guided-tour/automation/simple-helm/01-create-component/config.yaml) 
in the guided tour.

### Data flow

To allow a dataflow between the helm deployments, each item definition has 

```yaml
items:
  my-first-item:
    ...
    exports:
      token:
        schema:
          type: string
        fromResource:
          apiVersion: v1
          kind: Secret
          name: test-secret
          isNamespaced: true
          # namespace: example #optional: if isNamespaced==true and namespace is not set, use the release namespace
        jsonPath: .data.token
```


### Further features

- Exporting values
- Using exports of other items in the `additionalValues` (predefined variable `$imports`)
- Readiness checks
- Timeout
- Atomic (helm)


## Command to generate an Installation and Targets

```shell
landscaper-cli installation create [SETTINGS_FILE_PATH] [CONFIG_FILE_PATH] [RESULT_DIRECTORY_PATH]
```
