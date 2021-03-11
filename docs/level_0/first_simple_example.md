# First simple Example

In this example we describe how to deploy an nginx with the help of Landscaper.

We describe how to develop a component containing a Component Descriptor and a Blueprint with a DeployItem.
The Blueprint is a collection of DeployItems. In our case it contains exactly one DeployItem which specifies how to 
deploy the nginx as a helm chart.

## Deploy Item

We want to deploy an nginx via the helm chart [nginx ingress](https://github.com/kubernetes/ingress-nginx/tree/master/charts/ingress-nginx).
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
├── blueprint
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
[here](https://eu.gcr.io/sap-gcp-cp-k8s-stable-hub/examples/landscaper/docs/blueprints/github.com/gardener/landscaper/first-example).
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
    - baseUrl: "eu.gcr.io/sap-gcp-cp-k8s-stable-hub/examples/landscaper/docs"
      type: ociRegistry
  resources:
    - type: blueprint
      name: first-example-blueprint
      version: v0.1.0
      relation: local
      access:
        type: ociRegistry
        imageReference: eu.gcr.io/sap-gcp-cp-k8s-stable-hub/examples/landscaper/docs/blueprints/github.com/gardener/landscaper/first-example:v0.1.0
    - type: helm
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

We have uploaded the Component Descriptor into an OCI registry. You find it
[here](https://eu.gcr.io/sap-gcp-cp-k8s-stable-hub/examples/landscaper/docs/component-descriptors/github.com/gardener/landscaper/first-example).
You can upload it by yourself with the help of the Landscaper CLI command
[landscaper-cli components-cli component-archive remote push](https://github.com/gardener/landscapercli/blob/master/docs/reference/landscaper-cli_components-cli_component-archive_remote_push.md).

## Installation

Until now, we only created a reusable component and uploaded it to an OCI registry. Here we show
how to use this to install the nginx in some kubernetes cluster. Therefore, we create a custom 
resource of kind `Installation` which references the Component Descriptor and the contained Blueprint and deploy it to 
a cluster watched by Landscaper. This cluster is usually not the same as the cluster where you want to install the nginx. 

Remark: If you want to set up your own experimental landscaper and OCI registry you find a detailed description 
[here](https://github.com/gardener/landscapercli/blob/master/docs/commands/quickstart/install.md).

Before we develop the `Installation` we need a small preparation step. The nginx will be deployed in the namespace 
`first-example` on the target cluster. This was specified in the DeployItem. The helm deployment will not create this 
namespace automatically. We must do this manually with the following command, using the kubeconfig of the 
target cluster.

```
kubectl create namespace first-example
```

Now let's come back to our `Installation` you see here.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: demo
  namespace: demo
spec:
  componentDescriptor:
    ref:
      repositoryContext:
        type: ociRegistry
        baseUrl: eu.gcr.io/sap-gcp-cp-k8s-stable-hub/examples/landscaper/docs
      componentName: github.com/gardener/landscaper/first-example
      version: v0.1.0

  blueprint:
    ref:
      resourceName: first-example-blueprint

  imports:
    targets:
      - name: target-cluster
        target: "#my-cluster"
```

The Installation references the Component Descriptor. The Blueprint can be located by its resource name in the 
Component Descriptor where you also find its OCI address.

You remember the Blueprint has specified an import parameter `target-cluster`. The Installation defines how to retrieve 
the value for this parameter. In our case the value is a custom resource of kind `Target` with name `my-cluster`.

## Target

After the creation of the Installation, we can check its status. 

```
kubectl get installation -n demo demo

NAME   PHASE                 CONFIGGEN   EXECUTION   AGE
demo   PendingDependencies               demo        58m
```

You see the `Installation` get stuck in phase `PendingDependencies` because not all required import data are available yet.
The Installation will remain in this phase, as long as the Target with name `my-cluster` does not exist.
To fix this, we create the following Target on the same cluster and in the same namespace as the Installation. The  
Target contains the kubeconfig for the target cluster.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  name: my-cluster
  namespace: demo
spec:
  config:
    kubeconfig: |                     
      apiVersion: v1
      kind: Config
      ...
  type: landscaper.gardener.cloud/kubernetes-cluster
```

Now, after some time Landscaper installs the nginx on the target cluster and switches to the state of the
`Installation` to `Succeeded`.
