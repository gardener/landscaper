# Deploy applications via Landscaper using manifest deployer

- [Deploy applications via Landscaper using manifest deployer](#deploy-applications-via-landscaper-using-manifest-deployer)
  - [Introduction](#introduction)
  - [Preparation of work environment](#preparation-of-work-environment)
    - [Structure of demo material](#structure-of-demo-material)
    - [Install the Landscaper together with an OCI registry](#install-the-landscaper-together-with-an-oci-registry)
  - [Deploy applications via Landscaper](#deploy-applications-via-landscaper)
    - [1. Push demo image content into OCI registry](#1-push-demo-image-content-into-oci-registry)
    - [2. Develop Landscaper artifacts](#2-develop-landscaper-artifacts)
    - [3. Push Component Archive to OCI Registry](#3-push-component-archive-to-oci-registry)
    - [4. Deploy the application](#4-deploy-the-application)

## Introduction

This tutorial describes how to use the Landscaper manifest deployer for installing an application into a Kubernetes cluster.

The demo is ordered into the following activities:

  - [1. Push demo image content into OCI registry](#1-push-demo-image-content-into-oci-registry)
  - [2. Develop Landscaper artifacts](#2-develop-landscaper-artifacts)
  - [3. Push Component Archive to OCI Registry](#3-push-component-archive-to-oci-registry)
  - [4. Deploy the application](#4-deploy-the-application)

![alt text](README-deployment-4-steps.svg "Deployment with Landscaper in four steps")

## Preparation of work environment

Before you start, you need a working demo environment. If you do not have one, please refer to ["Prepare demo environment"](prepare-demo-environment.md).

First, download [demo material from this repository]([/](https://download-directory.github.io/?url=https%3A%2F%2Fgithub.com%2Fgardener%2Flandscaper%2Ftree%2Fmaster%2Fdocs%2Fgettingstarted%2fmanifest-deployer-tutorial)). You will edit some of the downloaded files later.

### Structure of demo material

The folder demo content contains a sample OCI image.
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
├── demo-content
│   └── hello.tar
├── manifests
│   ├── installation.yaml
│   ├── my-deployment.yaml
│   ├── my-secret.yaml
│   └── my-target.yaml
├── prepare-demo-environment.md
└── README.md
```

| Demo environment | Productive environment |
| ------------- |-------------|
| For demo purposes we will use a simplified setup with one cluster, which contains all necessary parts. | In a productive environment could be separated. |
| ![alt text](README-env-demo.svg "demo all-in-one cluster") | ![alt text](README-env-productice.svg "One demo cluster and other possible setup")|

### Install the Landscaper together with an OCI registry

The landscaper-cli provides a convenient quick start option for installing the Landscaper plus an OCI registry:

```bash
export OCI_USER=oci-user
export OCI_PASSWD='PSST!DONOTTELL!'

landscaper-cli quickstart install \
  --kubeconfig ~/.kube/config-demo.yaml \
  --install-oci-registry \
  --install-registry-ingress \
  --registry-username $OCI_USER \
  --registry-password $OCI_PASSWD
```

A successful installation should look like this:

``` text
Landscaper installation succeeded!

The OCI registry can be accessed via the URL https://o.ingress.p.democlust.shoot.k8s.myprovider.com
It might take some minutes until the TLS certificate is created
```

Remember, you need this URL to access the OCI registry later on:

``` bash
export OCI_REGISTRY=o.ingress.p.democlust.shoot.k8s.myprovider.com
```

Check if all needed components of Landscaper is available:

``` bash
kubectl get pods -n landscaper --kubeconfig ~/.kube/config-demo.yaml
```

The installation should contain

- Landscaper
- Landscaper webhooks
- Container deployer
- Helm deployer
- Manifest deployer
- OCI Registry

```text
NAME                                                    READY   STATUS    RESTARTS   AGE
container-default-container-deployer-5c6as449f9-dz4x6   1/1     Running   0          25s
helm-default-helm-deployer-7cd3fd9797-5c5rk             1/1     Running   0          41s
landscaper-654cbc4568-x6bs7                             1/1     Running   0          53s
landscaper-webhooks-58995rff45-htwcf                    1/1     Running   0          53s
manifest-default-manifest-deployer-7fbc87ef79-xpfnq     1/1     Running   0          31s
oci-registry-6654f55648-wg7bq                           1/1     Running   0          60s

```

Check availability of OCI registry:

``` bash
curl --location --request GET https://$OCI_REGISTRY/v2/_catalog -u "$OCI_USER:$OCI_PASSWD"
```

The curl should return this:

```text
{"repositories":[]}
```

## Deploy applications via Landscaper

### 1. Push demo image content into OCI registry

For this tutorial an OCI image is provided. The image contains a dummy application just for demonstration purposes. It does nothing, just keeps the container running.

The image is located at /manifest-deployer-tutorial/demo-content/hello.tar.
For the next steps, dockerd must be up and running in demo environment.

First, load the OCI image into local docker registry:

```bash
docker load --input ./demo-content/hello.tar
```

Then, push the OCI image into the OCI registry running inside the cluster.

``` bash
docker login -p $OCI_PASSWD -u $OCI_USER $OCI_REGISTRY

docker tag hello:v0.1.0 $OCI_REGISTRY/hello:v0.1.0

docker push $OCI_REGISTRY/hello:v0.1.0

curl --location --request GET https://$OCI_REGISTRY/v2/_catalog -u "$OCI_USER:$OCI_PASSWD"
```

### 2. Develop Landscaper artifacts

Landscaper artifacts are typically organized in a component archive as depicted below:

``` text
component-archive/
├── blueprint
│   ├── blueprint.yaml
│   └── deploy-execution.yaml
├── component-descriptor.yaml
└── resources.yaml
```

**[component-descriptor.yaml](https://github.com/gardener/landscaper/blob/master/docs/concepts/Glossary.md#component-descriptor)**: The component-descriptor.yaml is the BOM keeping track of all resources for a specific application with a specific version number. Under the definition of resources, things like container images, Landscaper blueprints, and helm charts are usually listed.

This is the pre-generated almost empty component descriptor. It already has a name and a version. You need to replace the placeholder `<OCIURL>` with the OCI registry URL obtained in the section "Install the Landscaper together with an OCI registry", e.g. "o.ingress.p.democlust.shoot.k8s.myprovider.com"

``` yaml
component:
  componentReferences: []
  name: test.net/test
  provider: internal
  repositoryContexts:
  - baseUrl: <OCIURL>
    componentNameMapping: urlPath
    type: ociRegistry
  resources: []
  sources: []
  version: v0.1.0
meta:
  schemaVersion: v2
```

**resources.yaml**: The resource.yaml contains a list of resource definitions for e.g. OCI images and Blueprints. The resources needed by our application are an OCI image containing the sample application and a blueprint describing the Landscaper deployment. Have a look at the resources and their yaml definition in /manifest-deployer-tutorial/resources.yaml. Later, it is shown how to update the component-descriptor.yaml with the resources specified in resources.yaml using the component-cli tool.

**[blueprint](https://github.com/gardener/landscaper/blob/master/docs/concepts/Glossary.md#blueprint)**: This folder contains the blueprint.yaml. The Blueprint specifies any import parameters necessary for a Landscaper deployment, it specifies what kind of deployment should be executed (deploy-execution.yaml), and it specifies if the deployment creates output via export parameters.

The application itself is not really of interest, it just keeps a container up and running. The container runs in a Pod, the  Pods has 3 Replicas, and all of this is specified in a Deployment manifest which you find in /manifest-deployer-tutorial/manifests/my-deployment.yaml. The other resource manifest we need for the application, is a Kubernetes resource of type Secret. This will allow Kubernetes to access the OCI registry which we deployed in the step "Install the Landscaper together with an OCI registry".

Let us go through the steps, to prepare our Landscaper manifest deployment:

1. In the component-descriptor.yaml (/manifest-deployer-tutorial/component-archive/), replace the `<OCIURL>` placeholder with the OCI registry URL as obtained in section "Install the Landscaper together with an OCI registry". E.g. replace `<OCIURL>` with o.ingress.p.democlust.shoot.k8s.myprovider.com.

2. In the resources.yaml (/manifest-deployer-tutorial/component-archive/), replace the `<IMAGEURL>` placeholder with the container image URL used in section "Push demo image content into OCI registry", to push the sample image into our OCI registry. E.g. replace `<IMAGEURL>` with o.ingress.p.democlust.shoot.k8s.myprovider.com/hello.

3. In deploy-excution.yaml (/manifest-deployer-tutorial/component-archive/blueprint/), replace the `<DOCKERCONFIG>` placeholder with a base64 encoded version of dockers config.json. You already did a docker login into your OCI registry, right?

    ``` bash
    cat ~/.docker/config.json | base64 --wrap=0
    ```

4. Add the resources defined in resource.yaml to component-descriptor.yaml by executing

    ``` bash
    component-cli component-archive resources add ./component-archive ./component-archive/resources.yaml
    ```

5. Verify updated content of component-descriptor. You need to replace the placeholder `<OCIURL>` with the OCI registry URL obtained in the section "Install the Landscaper together with an OCI registry", e.g. "o.ingress.p.democlust.shoot.k8s.myprovider.com"

    ``` yaml
    component:
      componentReferences: []
      name: test.net/test
      provider: internal
      repositoryContexts:
      - baseUrl: <OCIURL>
        componentNameMapping: urlPath
        type: ociRegistry
      resources:
      - access:
          imageReference: <OCIURL>/hello:v0.1.0
          type: ociRegistry
        name: hello
        relation: external
        type: ociImage
        version: v0.1.0
      - access:
          filename: sha256:a250692a6ca416d7d9442e22597eca9f2d5f0ab3dd2af231733ccdc3fb48820a
          mediaType: application/vnd.gardener.landscaper.blueprint.v1+tar+gzip
          type: localFilesystemBlob
        name: blueprint
        relation: local
        type: blueprint
        version: v0.1.0
      sources: []
      version: v0.1.0
    meta:
      schemaVersion: v2
    ```

6. Transform component-archive into Component Transport Format for our OCI registry:

    ``` bash
    component-cli ctf add ./transport.tar --component-archive ./component-archive/ --format tar
    ```

### 3. Push Component Archive to OCI Registry

Upload component-archive into our OCI registry:

``` bash
component-cli ctf push ./transport.tar
```

Verify content of OCI registry:

``` bash
curl --location --request GET https://$OCI_REGITRY/v2/_catalog -u "$OCI_USER:$OCI_PASSWD"
```

The curl should return the following:

``` text
{"repositories":["component-descriptors/test.net/test","hello"]}
```

### 4. Deploy the application

Lets take a quick summary of what has been done so far. We installed the Landscaper plus an OCI registry into a Kubernetes cluster. We pushed the OCI image of our hello application into the OCI registry. We developed the component-descriptor which describes all resources needed for deploying the application, we modified the blueprint which describes what needs to be done to deploy the application. These artifacts were transformed into an OCI artifact and pushed into the OCI registry.

Now, we need to tell Landscaper to pick up the artifacts from the OCI registry and execute the deployment. To achieve this, we need to develop two Kubernetes custom resources.

1. The Target resource. This tells the Landscaper **where** it should deploy the application to. Landscaper currently supports only one target type, a kubernetes cluster.

    ``` yaml
    apiVersion: landscaper.gardener.cloud/v1alpha1
    kind: Target
    metadata:
      creationTimestamp: null
      name: my-cluster
      namespace: example
    spec:
      config:
        kubeconfig: |+
        <KUBECONFIG>

      type: landscaper.gardener.cloud/kubernetes-cluster
    ```

    Either replace the `<KUBECONFIG>` placeholder with your clusters kubeconfig, or use the landscaper-cli tool (recommended), to create the Target manifest:

    ```bash
    landscaper-cli targets create kubernetes-cluster --name my-cluster --namespace example --target-kubeconfig ~/.kube/config-demo.yaml > ./my-target.yaml
    ```

2. The Installation resource. While the Blueprint provided a specification of all Import parameters, the Installation provides concrete values for the Imports. Furthermore, the Installation makes a connection to the component-descriptor associated with the application we are going to deploy. The below Installation specifies the component-descriptor, it specifies which blueprint it shall use (a component-descriptor can contain more than one blueprint). As for the Imports, there is just one parameter which needs to be initialized. There is an Import of type target with the name target. This is the same Target resource, we created in step 1. You find a sample Installation in /manifests/Installation.yaml. Replace the `<OCIURL>` placeholder with the OCI registry URL you created in section "Install the Landscaper together with an OCI registry".

    ``` yaml
    apiVersion: landscaper.gardener.cloud/v1alpha1
    kind: Installation
    metadata:
      name: manifest-demo
      namespace: example
    spec:
      blueprint:
        ref:
          resourceName: blueprint

      componentDescriptor:
        ref:
          componentName: test.net/test
          repositoryContext:
            baseUrl: <OCIURL>
            type: ociRegistry
          version: v0.1.0

      imports:
        targets:
          - name: cluster
            target: '#my-cluster'
    ```

3. Create **example** namespace

    ``` bash
    kubectl create namespace example
    ```

4. Apply the custom resources to your cluster:

    ``` bash
    kubectl apply -f ./my-target.yaml
    kubectl apply -f ./installation.yaml
    ```

5. Verify the status of the deployment:

    ``` bash
    landscaper-cli installations inspect manifest-demo -n example
    ```

    The final status should look like this:

    ``` bash
    >$ landscaper-cli installations inspect manifest-demo -n example
    [✅ Succeeded] Installation manifest-demo
        └── [✅ Succeeded] DeployItem manifest-demo-default-deploy-item-ncflh
    ```

This is the end of this tutorial.
