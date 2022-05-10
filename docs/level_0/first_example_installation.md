# First Example Installation

In this example we describe how to deploy your first component with Landscaper. The component defines an nginx helm chart deployment. Note that this example uses an already existing Component Descriptor and its Blueprint, so it is meant to give you a way to quickly deploy something with the Landscaper and look at the deployed resources in your cluster.

To try out this example by yourself, you need to install Landscaper (see [here](../gettingstarted/install-landscaper-controller.md)) in a cluster.

## Step 1 - Create and apply the Target Custom Resource

We want to deploy the nginx ingress-controller into a target cluster. Landscaper needs the access information for this cluster to execute 
the deployment. Therefore, we have to create a custom resource of type _Target_ in the namespace _demo_ in the cluster watched 
by Landscaper (which in this example is the cluster you have used to install the Landscaper):

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

Instead of creating this Target resource manually, you can also use the Landscaper CLI command [landscaper-cli target create](https://github.com/gardener/landscapercli/blob/master/docs/commands/targets/create.md)
to create the Target more comfortably.

After you have created the Target custom resource, you need to apply it to the Landscaper cluster:
```
kubectl apply -f _your_target_cr_filename
```

## Step 2 - Create and apply the K8s namespace for the deployment in the target cluster

The nginx ingress-controller will be deployed in a namespace `first-example` on the target cluster. The component **will not** create 
this namespace automatically. We must do this manually with the following command, using the kubeconfig of the 
target cluster:
```
kubectl create namespace first-example
```

## Step 3 - Create and apply the Installation custom resource

To install the nginx ingress-controller with the Landscaper, we have to finally create a custom resource of kind `Installation`. Such Ìnstallation`custom resources are watched by the Landscaper controller and triggers the installation as described by the Blueprint, which is located within the specified Component Descriptor.

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
      version: v0.1.3

  blueprint:
    ref:
      resourceName: first-example-blueprint

  imports:
    targets:
      - name: target-cluster
        target: "my-cluster"
```

The Installation references the [Component Descriptor](./basic_concepts.md#blueprint-component-and-component-descriptor) 
of the existing component `github.com/gardener/landscaper/first-example`. You find it
[here](https://eu.gcr.io/gardener-project/landscaper/tutorials/components/component-descriptors/github.com/gardener/landscaper/first-example).

The specified [Blueprint](./basic_concepts.md#blueprint) can be located by its resource name in the 
Component Descriptor. The Blueprint contains the specification of the nginx deployment. 

The Blueprint has an import parameter `target-cluster` to get the access data for the target cluster. 
The Installation sets the value of this parameter to the name `my-cluster` of the Target resource, we have created 
in Step 1. 

Now we have to _kubectl apply_ this `Installation` custom resource to the same cluster and namespace as the Target resource from above. After some time, 
Landscaper installs the nginx on the target cluster and switches the state of the `Installation` to `Succeeded`.

You could check this in your Landscaper cluster:

```shell
k get inst -n demo demo                            
NAME   PHASE        CONFIGGEN    EXECUTION   AGE
demo   Succeeded                             2m
```

If there is already another ingress controller installed on the target cluster, the installation might fail due to a conflict.

If you want to know how the component of this example is created go to the [first example component](./first_example_component.md) page. 
If you want to know more about the concepts go to the [basic concepts](./basic_concepts.md) page.
