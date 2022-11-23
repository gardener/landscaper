# Manifest Deployer Example

For prerequisites see [here](../../README.md#prerequisites-and-basic-definitions).

The Landscaper offers different deployers: 

- the [Helm Deployer](../../../deployer/helm.md)
- the [Kubernetes Manifest Deployer](../../../deployer/manifest.md), 
- and the [Container Deployer](../../../deployer/container.md).

We have already used the Helm deployer in the Hello World Example to create a ConfigMap on the target cluster.
In the present example we show how the same task can be achieved with the Kubernetes manifest deployer.
The Kubernetes manifest deployer is suitable if you want to deploy some Kubernetes manifests without extra building a 
Helm chart for them. The Kubernetes manifests to be deployed are directly included in the blueprint of the Installation.

Let's have a look into the blueprint of the [Installation](installation/installation.yaml). It contains one DeployItem:

```yaml
                deployItems:
                  - name: default-deploy-item
                    type: landscaper.gardener.cloud/kubernetes-manifest
                    config:  
                      manifests:
                        - manifest:
                            apiVersion: v1
                            kind: ConfigMap
                            metadata:
                              name: hello-world
                              namespace: example
                            data:
                              testData: hello
```

The type `landscaper.gardener.cloud/kubernetes-manifest` tells Landscaper that the manifest deployer 
should be used to process the DeployItem.
A DeployItem also contains a `config` section, whose structure depends on the type. In case of the manifest deployer,
the `config` section contains the list of Kubernetes manifest that should be applied to the target cluster. 
In the present example, this list contains the Kubernetes manifest of the ConfigMap that we want to create. 


## Procedure

1. Insert in file [target.yaml](installation/target.yaml) the kubeconfig of your target cluster.

2. On the Landscaper resource cluster, create namespace `example` and apply 
   the [target.yaml](installation/target.yaml) and the [installation.yaml](installation/installation.yaml):
   
   ```shell
   kubectl create ns example
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml>
   ```

3. Wait until the Installation is in phase `Succeeded` and check that the ConfigMap was created.
