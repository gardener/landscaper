# First Example Installation

In this example we describe how to deploy your first component with Landscaper. The component defines an nginx helm 
chart deployment. 

To try out this example by yourself you need to install Landscaper (see [here](../gettingstarted/install-landscaper-controller.md)).

## Step 1

We want to deploy the nginx ingress-controller on some target cluster. Landscaper needs the access information for this cluster to execute 
the deployment. Therefore, we create a custom resource of type Target in some namespace `demo` on the cluster watched 
by Landscaper:

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  name: my-cluster
  namespace: demo
spec:
  type: landscaper.gardener.cloud/kubernetes-cluster
  config:
    kubeconfig: |                     
      apiVersion: v1
      kind: Config
      ...
```

The field `spec.config.kubeconfig` must contain the kubeconfig of the target cluster.

You can use the Landscaper CLI command [landscaper-cli target create](https://github.com/gardener/landscapercli/blob/master/docs/commands/targets/create.md)
to create the Target more comfortably.

## Step 2

The nginx ingress-controller will be deployed in a namespace `first-example` on the target cluster. The component **will not** create 
this namespace automatically. We must do this manually with the following command, using the kubeconfig of the 
target cluster.

```
kubectl create namespace first-example
```

## Step 3

To install the nginx we will create a custom resource of kind `Installation`.

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
        baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components
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

The Installation references the [Component Descriptor](./basic_concepts.md#blueprint-component-and-component-descriptor) 
of the existing component `github.com/gardener/landscaper/first-example`. You find it
[here](https://eu.gcr.io/gardener-project/landscaper/tutorials/components/component-descriptors/github.com/gardener/landscaper/first-example).

The specified [Blueprint](./basic_concepts.md#blueprint) can be located by its resource name in the 
Component Descriptor. The Blueprint contains the specification of the nginx deployment. 

The Blueprint has an import parameter `target-cluster` to get the access data for the target cluster. 
The Installation sets the value of this parameter to the name `my-cluster` of the Target resource, we have created 
in Step 1. 

Now we deploy the `Installation` in the same cluster and namespace as the Target resource from above. After some time 
Landscaper installs the nginx on the target cluster and switches to the state of the `Installation` to `Succeeded`.

You could check this in your Landscaper cluster:

```shell
k get inst -n demo demo                            
NAME   PHASE        CONFIGGEN    EXECUTION   AGE
demo   Succeeded                             2m
```

If you want to know how the component of this example is created go to the [first example component](./first_example_component.md) page. 
If you want to know more about the concepts go to the [basic concepts](./basic_concepts.md) page.
