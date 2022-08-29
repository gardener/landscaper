# Update applications via Landscaper using manifest deployer tutorial

**Introduction**

This tutorial describes how to use the Landscaper manifest deployer for updating your application in a Kubernetes cluster.

The example is based on the component created by the [manifest deployer tutorial](README.md).

The demo is ordered into the following activities:

- [Update applications via Landscaper using manifest deployer tutorial](#update-applications-via-landscaper-using-manifest-deployer-tutorial)
  - [1. Build new version of component](#1-build-new-version-of-component)
  - [2. Update resources with new version](#2-update-resources-with-new-version)
  - [3. Create and push updated Landscaper artifacts to OCI Registry](#3-create-and-push-updated-landscaper-artifacts-to-oci-registry)
  - [4. Deploy the updated application](#4-deploy-the-updated-application)

**Structure of demo material**

The folder hello contains a sample OCI image which we will update.
The folder manifests contains all Kubernetes manifests.
The folder component-archive contains all resources needed for building Landscaper deployable component.

``` text
./manifest-deployer-tutorial/
├── component-archive
│   ├── blueprint
│   │   ├── blueprint.yaml
│   │   └── deploy-execution.yaml
│   ├── component-descriptor.yaml
│   └── resources.yaml
├── hello
│   ├── Dockerfile
│   └── index.html
├── manifests
│   ├── installation.yaml
│   ├── my-deployment.yaml
│   ├── my-secret.yaml
│   └── my-target.yaml
├── prepare-dev-environment.md
├── README.md
└── update-application.md
```

## 1. Build new version of component

First, to see the changes we will do, we just adapt the index.html of the sample OCI with the new version:

```html
  <pre>
    hello app version 0.2.0
    OCI artifact version 0.1.1
  </pre>
```

Build the updated sample OCI image and push into local docker which registry running inside the cluster:

```bash
docker build . -t $OCI_REGISTRY/hello:v0.2.0
```

Check the contents of the registry with 

```bash
docker images $OCI_REGISTRY/hello
```

The output should be like

```text
REPOSITORY                                                        TAG       IMAGE ID       CREATED         SIZE
o.ingress.demo.hubtest.shoot.canary.k8s-hana.ondemand.com/hello   v0.2.0    123b387793dd   5 seconds ago   1.24MB
o.ingress.demo.hubtest.shoot.canary.k8s-hana.ondemand.com/hello   v0.1.0    123f943914b8   3 days ago    1.24MB
```

## 2. Update resources with new version

Landscaper artifacts are typically organized in a component archive as depicted below:

``` text
component-archive/
├── blueprint
│   ├── blueprint.yaml
│   └── deploy-execution.yaml
├── component-descriptor.yaml
└── resources.yaml
```

**resources.yaml**: The resource.yaml contains a list of resource definitions for e.g. OCI images and Blueprints. The resources needed by our application are an OCI image containing the sample application and a blueprint describing the Landscaper deployment. Here we update to the newer version.

```yaml
  imageReference: >-
    o.ingress.demo.hubtest.shoot.canary.k8s-hana.ondemand.com/hello:v0.2.0
```

## 3. Create and push updated Landscaper artifacts to OCI Registry

Convert the updated component located in ./component-archive/ into the Component Transport Format (CTF/ just a compressed tar of the folder structure) and add it to a component archive (just a tar of one or more components in CTF) located here ./transport.tar:

```bash
# if already exists, delete in advance
rm ./transport.tar

component-cli component-archive ./component-archive/ ./transport.tar -r ./component-archive/resources.yaml --component-version "v0.1.1"
```

Here we use a path version for component-archive, although the sample OCI image had a minor version update, just to see what all the version tags mean, and upload it into our OCI registry:

```bash
component-cli ctf push ./transport.tar
```

Verify content of OCI registry:

```bash
curl --location --request GET https://$OCI_REGISTRY/v2/_catalog -u "$OCI_USER:$OCI_PASSWD"
```

The curl should return the following:

```text
{"repositories":["component-descriptors/test.net/test","hello"]}
```

## 4. Deploy the updated application

Let's take a quick summary of what has been done so far. We created a new version of the OCI image of our hello application and pushed into the OCI registry (as v0.2.0). We changed the reference in resources.yaml to the new OCI image of our hello app. We transformed it into a new component archive (v0.1.1) and uploaded as an OCI artifact into the OCI registry.

Now, we need to tell Landscaper to pick up the artifacts from the OCI registry and execute the deployment. To achieve this, we simply change the version of our new OCI artifact in the installation resource in /manifests/installation.yaml. Replace the `componentDescriptor/version` to `v0.1.1`:

```yaml
      version: v0.1.1
```

Now apply the custom resources to your cluster:

```bash
kubectl apply -f manifests/installation.yaml --kubeconfig ~/.kube/config-demo.yaml
```

Verify the status of the deployment:

``` bash
landscaper-cli installations inspect manifest-demo -n example
```

The final status should look like this:

``` bash
>$ landscaper-cli installations inspect manifest-demo -n example
[✅ Succeeded] Installation manifest-demo
    └── [✅ Succeeded] DeployItem manifest-demo-default-deploy-item-5w7jv
```

Check again the deployments:

After some time the deployment and pod(s) of your example should appear:

```bash
kubectl get deployments --kubeconfig ~/.kube/config-demo.yaml -n example
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
hello-deployment   1/1     1            1           12m


kubectl get pods -n example --kubeconfig ~/.kube/config-demo.yaml
NAME                                READY   STATUS    RESTARTS   AGE
hello-deployment-6f8b8ff985-4gcr4   1/1     Running   0          8m
```

This is the end of this tutorial.
