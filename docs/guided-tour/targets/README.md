---
title: Constructing a Target Resource
sidebar_position: 50
---

# Constructing a Target Resource

Suppose you want to use the Landscaper to deploy an application on some kubernetes cluster, the so-called *target cluster*.
You need to create a `Target` custom resource which contains a kubeconfig for the target cluster.

If your target cluster is a Gardener shoot cluster, you typically have an oidc / gardenlogin kubeconfig for the target cluster.
It is **not** possible to use such a kubeconfig in a `Target` custom resource.
In the following we describe how to build a kubeconfig which you can use in a `Target`.
It will be based on a ServiceAccount token.


### Define Names

First, let's define some names for the resources we are going to create. You can choose your own names.

```shell
export serviceaccount_name=admin
export serviceaccount_namespace=guided-tour
export clusterrolebinding_name=guided-tour-admin
```


### Create ServiceAccount 

On the target cluster, create the Namespace and ServiceAccount:

```shell
kubectl create namespace ${serviceaccount_namespace}
kubectl create serviceaccount -n ${serviceaccount_namespace} ${serviceaccount_name}
```


### Create ClusterRoleBinding

Create a ClusterRoleBinding which binds the ServiceAccount to the ClusterRole `cluster-admin`. 
You can choose another ClusterRole. However, it must grant enough permissions to deploy applications to your target cluster:

```shell
export "clusterrolebinding_name=${clusterrolebinding_name}"
export "serviceaccount_name=${serviceaccount_name}"
export "serviceaccount_namespace=${serviceaccount_namespace}"
inputFile="${COMPONENT_DIR}/installation/installation-upg.yaml.tpl"
envsubst < ${inputFile} | kubectl apply -f -
```

Alternatively, you can manually apply the [ClusterRoleBinding manifest](./resources/clusterrolebinding.yaml.tpl) 
(replace the variables).


### Create ServiceAccount Token

Create a token for the ServiceAccount:

```shell
token=$(kubectl create token -n ${serviceaccount_namespace} ${serviceaccount_name} --duration=7776000s)
```


### Build the Kubeconfig

Copy your kubeconfig, and replace the `users[].user` section (insert your token for the variable):

```yaml
apiVersion: v1
kind: Config

...

users:
  - name: ...
    user:
      token: <insert your token here>
```

Check whether you can access your target cluster with this kubeconfig.


### Create Target

We can now create a Target custom resource using the kubeconfig t

On the **resource cluster**, you can now create a `Target` custom resource containing the above kubeconfig:

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  name: my-cluster
  namespace: example
spec:
  type: landscaper.gardener.cloud/kubernetes-cluster
  config:
    kubeconfig: |
      apiVersion: v1                          # <-------------------------- replace with your kubeconfig
      kind: Config                            #
                                              #
      ...                                     #
                                              #
      users:                                  #
        - name: ...                           #
          user:                               #
            token:  <insert your token here>  #
```
