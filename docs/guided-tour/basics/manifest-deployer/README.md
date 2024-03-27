---
title: The Manifest Deployer
sidebar_position: 2
---

# Manifest Deployer Example

Let's have a closer look at the Manifest Deployer.

For prerequisites, see [here](../../README.md).

The Landscaper offers different deployers per default: 

- the [Helm Deployer](../../../deployer/helm.md)
- the [Kubernetes Manifest Deployer](../../../deployer/manifest.md), 
- and the [Container Deployer](../../../deployer/container.md).

We have already used the Helm deployer in the first Hello World Example to deploy a 
Helm Chart to create a ConfigMap on the target cluster.

In the current example, we will show how the same task can be achieved with the Kubernetes manifest deployer.
This deployer is great if you want to deploy some Kubernetes manifests without going the extra mile of building a Helm chart for these manifests. The Kubernetes manifests are directly included in the blueprint of the Installation.

Let's look at the blueprint of the [Installation](installation/installation.yaml). It contains one DeployItem:

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

The type `landscaper.gardener.cloud/kubernetes-manifest` tells Landscaper that the manifest deployer should be used to process the DeployItem. A DeployItem also contains a `config` section, and its structure depends on the DeployItem type. In case of the manifest deployer, the `config` section contains the list of Kubernetes manifests,which should be applied to the target cluster. In this example, the list contains one Kubernetes manifest for a ConfigMap.

## Procedure

1. Insert the kubeconfig of your target cluster into your [target.yaml](installation/target.yaml). 

2. On the Landscaper resource cluster, create the namespace `example` and apply your [target.yaml](installation/target.yaml) and [installation.yaml](installation/installation.yaml):
   
   ```shell
   kubectl create ns example
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml>
   ```

3. Wait until the Installation is in phase `Succeeded` and check that the ConfigMap was created.
