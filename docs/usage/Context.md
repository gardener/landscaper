# Context

A context is a configuration resource which could be referenced and used by different installations. It contains shared 
configuration data for installations. This information includes the location of the component descriptors as well as 
access data like credentials for the component descriptors and other (OCI) artifacts like images, blueprints etc. 

As you already know, an installation references a blueprint and a component descriptor. The component descriptor itself 
might reference further artifacts like OCI images, other component descriptors etc. With the information in the context,
the Landscapes knows the location of the components descriptors and possesses the required credentials to all OCI 
artefacts (including the component descriptors).

Remark: Be aware that all components descriptors must be located in the same repository context, e.g. if they are stored
in an OCI registry in a repository context `example.com/somePath`, it is assumed that all component descriptors are located
under `example.com/somePath/component-descriptors/`

A context can only be referenced by installations in the same namespace.

## Basic structure

A context object has the following structure.  It contains the repository context, registry pull secrets for accessing
resources stored in OCI registries, and additional information in the configurations section.


```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Context
metadata:
  name: example-context
  namespace: example-namespace

repositoryContext:
  type: ociRegistry
  baseUrl: "example.com"

registryPullSecrets: # additional pull secrets to access component descriptors and blueprints
- name: my-pullsecret

configurations: 
  yourKey: yourInfo
```

The repository context is usually the location where the component descriptors are stored in an OCI registry. For the 
example above it is expected that the component descriptors are stored under `example.com/component-descriptors/`.

These registry pull secrets are references to secrets in the same namespace as the context. It is expected that the 
secrets contain oci registry access credentials. These credentials are used by the Landscaper to access component 
descriptors, blueprints, images or even deployable artifacts like helm charts stored in an OCI registry.

How to create registry pull secrets is described
[here](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/). They typically look as
follows:

```
apiVersion: v1
kind: Secret
metadata:
  name: my-pullsecret
  namespace: example-namespace
data:
  .dockerconfigjson: authenticationData
type: kubernetes.io/dockerconfigjson
```

**Example**:

This example describes how to create a secret with access data to the  Google Container Registry. First, you need a 
service account with read permissions for your registry. Then you create a service account key and download the 
corresponding service account key file ([see](https://cloud.google.com/iam/docs/creating-managing-service-account-keys)). 

Finally, you create the authentication data for the secret according to the following example. 

```
{
  "auths":{
    "eu.gcr.io/somePath":{"auth":"base64 encoded content of the service account key file"}
  }
}
```

`eu.gcr.io/somePath` defines that all OCI artefacts stored at a location starting with this URL are fetched using the 
specified access data of the `auth` entry. Of course, you need to use your URL here. If your resources are located
in different OCI registries, you could add several URLs together with the appropriate access data. 

## Installation with Context Reference

An installation could reference a context object as outlined here:

```
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: example-name
  namespace: example-namespace

spec:
  blueprint:
    ref:
      resourceName: someBlueprint

  context: example-context

  componentDescriptor:
    ref:
      componentName: yourDomain/your/component/name
      version: v0.1.0
```

When this installation is deployed, the Landscaper locates the component descriptor by the repository context in the
context object concatenated with `component-descriptors` and the name of the component descriptor:

    `example.com/component-descriptors/yourDomain/your/component/name`

For all referenced component descriptors the location is computed the same way. For all resources stored in protected
OCI registries, the Landscaper uses the registry pull secrets provided by the context object to get access to them.

## Default Context in the Landscaper Configuration

Just like kubernetes creates a default service account in every namespace, the Landscaper creates a default context object
in every namespace during startup. The default context can be configured in the landscaper configuration 
`.controllers.context.config.default` as described in this [example](../../examples/00-Landscaper-Configuration.yaml):

```yaml
apiVersion: config.landscaper.gardener.cloud/v1alpha1
kind: LandscaperConfiguration

controllers:
  context:
    config:
      default:
        disable: false
        excludeNamespaces: # optional
        - kube-system
        - ls-system
        repositoryContext: # define the default repository context for installations
          type: ociRegistry
          baseUrl: "myregistry.com/components"
```

If nothing is configured for the default context in the LandscaperConfiguration, empty default contexts are still 
created. In this situation you could modify these context objects manually. This is not possible if you have configured 
something in the LandscaperConfiguration because the responsible context controller replaces your manual 
modifications always with these settings.

If an installation has no context configured, the default context is used. 
