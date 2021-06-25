# First Example Component

In this example we describe how to develop the component that we have used in the
[First Example Installation](./first_example_installation.md).

The component contains a Component Descriptor and a Blueprint with one DeployItem. The DeployItem specifies how to 
deploy the nginx as a helm chart.

## Deploy Item

We want to deploy an nginx ingress-controller via the helm chart [nginx ingress](https://github.com/kubernetes/ingress-nginx/tree/master/charts/ingress-nginx).
Therefore, we specify a DeployItem in a yaml file with the following content.

```yaml
deployItems:
- name: nginx
  type: landscaper.gardener.cloud/helm                                                             # (1)
  target:
    name: {{ index .imports "target-cluster" "metadata" "name" }}                                  # (2)
    namespace: {{ index .imports "target-cluster" "metadata" "namespace" }}                        # (3)
  config:
    apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration

    chart:
      ref: "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v0.1.0"           # (4)

    updateStrategy: patch

    name: nginx                                                                                    # (5)
    namespace: first-example                                                                       # (6)
```

Since we want to deploy a helm chart, the type of the DeployItem is set to `landscaper.gardener.cloud/helm` in line (1).

The fields in lines (2) and (3) are not yet specified. They are bound to the import parameter `target-cluster` of the 
Blueprint, that we will define below. The target section defines the name and namespace of a kubernetes custom resource
containing the access data for the target cluster on which the nginx will be deployed. We leave it variable, in order 
to make the Deploy Item reusable. So it can be used to deploy the nginx on different clusters. 

Line (4) contains a reference to the helm chart for the nginx. The helm chart is stored in an OCI registry because 
Landscaper is currently not able to fetch helm charts from helm chart repositories.

Line (5) and (6) define the name of the nginx helm installation, and the namespace into which it will be deployed. In a 
real world scenario it would be a good idea to provide the value of the namespace as import data to get a better 
reusable component.

You find the DeployItem in file [deploy-execution-nginx.yaml](./resources/blueprint/deploy-execution-nginx.yaml).

## Blueprint

Next, we specify the Blueprint. The Blueprint consists of a [directory](./resources/blueprint) with the following 
structure:

```
└── blueprint
    ├── blueprint.yaml
    └── deploy-execution-nginx.yaml
```

The Blueprint contains a
[blueprint.yaml](./resources/blueprint/blueprint.yaml) with the following content, and the 
[deploy-execution-nginx.yaml](./resources/blueprint/deploy-execution-nginx.yaml) of the DeployItem. 
 
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

imports:                                                   # (1)
- name: target-cluster
  type: target
  targetType: landscaper.gardener.cloud/kubernetes-cluster 

deployExecutions:                                          # (2)
- name: nginx-execution 
  file: /deploy-execution-nginx.yaml
  type: GoTemplate
```

The `imports` section (1) defines the interface of the Blueprint. In this example, there is only one import parameter 
with name `target-cluster`. This parameter will be used to provide the access data to the target cluster where the 
nginx should be deployed to. The `target-cluster` parameter has a predefined type 
`landscaper.gardener.cloud/kubernetes-cluster`. A value of this type is the name of a custom resource `Target` 
containing the access data for a kubernetes cluster.

By means of the import parameter `target-cluster` the Blueprint becomes reusable, and you can deploy an nginx on different 
clusters.

The `deployExecutions` section (2) contains a reference to the file of the DeployItem that we have created in the 
previous step. 

We have uploaded the Blueprint into an OCI registry. You find the Blueprint OCI artifact 
[here](https://eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/first-example).
You can upload a Blueprint by yourself to another OCI registry with the help of the Landscaper CLI command 
[landscaper-cli blueprints push](https://github.com/gardener/landscapercli/blob/master/docs/reference/landscaper-cli_blueprints_push.md).

## Component Descriptor

Our component needs a Component Descriptor that contains the list of all required resources and how to access them.
Here, these are the Blueprint, and the helm chart of the nginx with their corresponding OCI references.

```yaml
component:
  componentReferences: []
  name: github.com/gardener/landscaper/first-example
  provider: internal
  repositoryContexts:
    - baseUrl: "eu.gcr.io/gardener-project/landscaper/tutorials/components"
      type: ociRegistry
  resources:
    - type: blueprint
      name: first-example-blueprint
      version: v0.1.0
      relation: local
      access:
        type: ociRegistry
        imageReference: eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/first-example:v0.1.0
    - type: helm.io/chart
      name: nginx-chart
      version: 0.1.0
      relation: external
      access:
        imageReference: eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v0.1.0
        type: ociRegistry
  sources: []
  version: v0.1.0
meta:
  schemaVersion: v2
```

We previously uploaded the Component Descriptor into an OCI registry. You can find it
[here](https://eu.gcr.io/gardener-project/landscaper/tutorials/components/component-descriptors/github.com/gardener/landscaper/first-example).
You can upload a Component Descriptor by yourself to another OCI registry with the help of the Landscaper CLI command
[landscaper-cli components-cli component-archive remote push](https://github.com/gardener/landscapercli/blob/master/docs/reference/landscaper-cli_components-cli_component-archive_remote_push.md).

We have now created a reusable component and uploaded it to an OCI registry. If you want to deploy this component on a
target cluster, see [First Example Installation](./first_example_installation.md).
