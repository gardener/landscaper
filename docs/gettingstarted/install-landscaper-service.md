# Landscaper Service

The motivation of the landscaper service is to host multiple isolated landscaper installations in a single Kubernetes cluster that can be used by different tenants.

## Architecture

An instance of the landscaper service is installed within a namespace of a _hosting cluster_.
The landscaper service is contained in the [landscaper-service](../../.landscaper/landscaper-service) component.

* The landscaper service component uses the `virtual garden` component as a component reference.
The virtual garden component installs an etcd, Kube API server and Kube controller manager. The API server creates a _virtual cluster_. The name virtual is used here, because, it is node-less and no deployments can be created within the virtual cluster. Its purpose is to create/manage landscaper resources within the virtual cluster.

* The `RBAC blueprint` installs the required Kubernetes RBAC resources in the virtual cluster.

* The `landscaper blueprint` installs the landscaper controller and the landscaper webhooks server.

* The `installation blueprint` contains the virtual garden, RBAC and landscaper blueprint as subinstallations and installs them in the correct order.

## Installation

The landscaper service can be installed by creating an installation of the `installation blueprint`.

### Target Cluster

The installation requires a target of type `landscaper.gardener.cloud/kubernetes-cluster` which specifies the hosting cluster into which the landscaper service will be installed. See [this documentation](../technical/target_types.md) on how to create a target resource.

### OCI Secrets

The OCI registry secrets can be specified as a Kubernetes secret.
The secret must contain secret data in the docker auth format.

Example:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-oci-secrets
type: Opaque
stringData:
  default: |
    {
      "auths": {
        "https://oci.acme.cloud": {
          "auth": "Zm9vOmJhcg=="
        }
      }
    }
```

### Installation

The landscaper service can be installed by creating an `installation` which references to the `installation-blueprint` of the `github.com/gardener/landscaper` component.
The following example shows the configuration of the landscaper service installation.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  annotations:
    # this annotation is required such that the installation is picked up by the Landscaper
    # it will be removed when processing has started
    landscaper.gardener.cloud/operation: reconcile
  name: landscaper-service
spec:
  componentDescriptor:
    ref:
      repositoryContext:
        type: ociRegistry
        baseUrl: eu.gcr.io/gardener-project/development
      componentName: github.com/gardener/landscaper/landscaper-service
      version: v0.14.0

  blueprint:
    ref:
      resourceName: installation-blueprint

  imports:
    targets:
    # The "hostingCluster" specifies the target Kubernetes cluster into which the landscaper service will be installed.
    # References a landscaper "target" resource.
    - name: hostingCluster
      target: "my-target-cluster"
    data:
    # The OCI registry secretes can be referenced as a Kubernetes secret.
    - name: ociSecrets
      secretRef:
        name: my-oci-secrets

  importDataMappings:
    # The "hostingClusterNamespace" is the namespace in the hosting cluster into which the landscaper service is getting installed.
    # Only one landscaper server per namespace can be installed.
    hostingClusterNamespace: landscaper-service
    # Specifies whether the "hostingClusterNamespace" is getting deleted before the installation.
    deleteHostingClusterNamespace: true
    # The "virtualClusterNamespace" is the namespace in the virtual cluster into which the landscaper resources are getting installed.
    virtualClusterNamespace: ls-system
    # The underlying infrastructure provider of the "hostingCluster" Kubernetes cluster.
    # Currently, supported values: "gcp", "aws", "alicloud"
    providerType: gcp
    # The DNS domain at which the landscaper service should be accessible.
    # Used for certificate generation.
    dnsAccessDomain: landscaper.acme.cloud

    # The landscaper registry configuration.
    registryConfig:
      cache:
        useInMemoryOverlay: false
      allowPlainHttpRegistries: false
      insecureSkipVerify: false
      # Specification/Reference to the oci secrets.
      secrets: (( ociSecrets ))

    # The landscaper deployment configuration.
    landscaperConfig:
      # Landscaper controller configuration.
      landscaper:
        # Optional: log verbosity
        verbosity: 2
        # Optional: the number of replicas to use for the landscaper controller deployment
        replicas: 1
        # Optional: the resource specification for the landscaper controller pods
        resources:
          requests:
            memory: "64Mi"
            cpu: "250m"
          limits:
            memory: "128Mi"
            cpu: "500m"
      # Landscaper webhooks server configuration
      webhooksServer:
        # Optional: the webhooks server service port to use
        servicePort: 9443
        # Optional: the number of replicas to use for the landscaper webhooks server deployment
        replicas: 1
      # The list of deployers that should be installed alongside the landscaper service.
      deployers:
        - container
        - helm
        - manifest
      # Optional: the configuration for the deployers
      # The configuration can be used to pass any deployer helm chart values.
      deployersConfig:
        # Config for the helm deployer.
        helm:
          verbosity: 2
        # Config for the container deployer.
        container:
          verbosity: 10
          replicas: 1
          resources:
          requests:
            memory: "64Mi"
            cpu: "250m"
          limits:
            memory: "128Mi"
            cpu: "500m"
        # Config for the manifest deployer.
        manifest:
          verbosity: 3
        # Additional deployers configuration.

  # The landscaper service exports two data objects.
  # "landscaperClusterEndpoint" is the API server endpoint at which the landscaper is available.
  # "landscaperClusterKubeconfig" is the Kubernets kubeconfig which can be used to connect to the API server.
  exports:
    data:
    - name: landscaperClusterEndpoint
      dataRef: landscaperClusterEndpoint
    - name: landscaperClusterKubeconfig
      dataRef: landscaperClusterKubeconfig
```

The installation exports two string data objects:
1. `landscaperClusterEndpoint` is the endpoint of the virtual cluster API server at which the landscaper available.
2. The `landscaperClusterKubeconfig` is a base 64 encoded kubeconfig file which can be used to connect to the virtual cluster API server as a landscaper user.
The landscaper user has permissions to create landscaper resources, secrets, configmaps and namespaces in the virtual cluster.
