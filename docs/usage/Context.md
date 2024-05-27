---
title: Context
sidebar_position: 3
---

# Context

## Definition

A context is a configuration resource which could be referenced and used by different installations. It contains shared 
configuration data for installations. This information includes the location of the component descriptors as well as 
access data like credentials for the component descriptors and other (OCI) artifacts like images, blueprints etc. 

As you already know, an installation references a blueprint and a component descriptor. The component descriptor itself 
might reference further artifacts like OCI images, other component descriptors etc. With the information in the context,
the Landscaper knows the location of the components descriptors and possesses the required credentials to all OCI 
artefacts (including the component descriptors).

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
  
ocmConfig:
  name: example-ocm-config

# DEPRECATED
repositoryContext:
  type: ociRegistry
  baseUrl: "example.com"

registryPullSecrets: # additional pull secrets to access component descriptors and blueprints
- name: my-pullsecret
    
configurations:
  config.mydeployer.mydomain.org: ... # custom configuration, not evaluated by landscaper
```

>**DEPRECATED:**  
> The `repositoryContext` has been deprecated and is superseded by the specification of resolvers in the `ocmConfig` as
> shown and explained below.

The repository context is usually the location where the component descriptors are stored in an OCI registry. For the 
example above it is expected that the component descriptors are stored under `example.com/component-descriptors/`.

The `ocmConfig` is a reference to a config map in the same namespace containing a key `.ocmconfig` with the 
corresponding value being ocm configuration data.
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ocm-config
  namespace: example
data:
  .ocmconfig: |
      type: generic.config.ocm.software/v1
      configurations:
        - type: ocm.config.ocm.software
          resolvers:
          - repository:
              type: OCIRegistry
              baseUrl: ghcr.io
              subPath: ocm-example-repo-1
            prefix: github.com/acme.org/component
            priority: 10
          - repository:
              type: OCIRegistry
              baseUrl: docker.io
              subPath: ocm-example-repo-2
            prefix: github.com/acme.org/referenced-component
            priority: 10
```

So this config map is a representation of the [ocm configfile](https://ocm.software/docs/cli-reference/help/configfile/) 
concept as a kubernetes API object. Consequently, you can test certain ocm configurations locally, using your ocm 
configfile (located at $HOME/.ocmconfig per default) and the ocm-cli and then copy the files contents into the config 
map under the key `.ocmconfig`.
The `resolvers` can be used to replace the `repositoryContext` specification in the Context object. This also allows to
specify multiple repositories. So, the component specified in the installation can reference a component located in
another repository. In the example above, a component called `github.com/acme.org/component` stored in 
`ghcr.io/ocm-example-repo-1` can have a reference to a component called `github.com/acme.org/referenced-component` 
stored in `docker.io/ocm-example-repo-2`. For further details, check the 
[ocm configfile documentation](https://ocm.software/docs/cli-reference/help/configfile/).

> **NOTE:**  
> The ocm configfile allows to influence almost all parts of the ocm tooling's behavior. While several of these features 
> work technically already, only the configuration of resolvers is officially supported for now. 

These registry pull secrets are references to secrets in the same namespace as the context. It is expected that the 
secrets contain oci registry access credentials. These credentials are used by the Landscaper to access component 
descriptors, blueprints, images or even deployable artifacts like helm charts stored in an OCI registry.

How to create registry pull secrets is described
[here](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/). They typically look as
follows:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-pullsecret
  namespace: example-namespace
data:
  .dockerconfigjson: authenticationData
type: kubernetes.io/dockerconfigjson
```

**Example for Google Container Registry**:

This example describes how to create a secret with access data to the Google Container Registry. You find more detailed
information [here](https://cloud.google.com/iam/docs/creating-managing-service-account-keys). First, you need a 
service account with read permissions for your registry. Then you create a service account key and download the 
corresponding service account key file to e.g. `~/json-key-file-from-gcp.json` 
([see](https://cloud.google.com/iam/docs/creating-managing-service-account-keys)). 

Finally, you create the secret with the authentication data with this command assuming your registry is located under
the domain `eu.gcr.io`: 

```
kubectl create secret docker-registry my-pullsecret \
  -n example-namespace \
  --docker-server=eu.gcr.io \
  --docker-username=_json_key \
  --docker-password="$(cat ~/json-key-file-from-gcp.json)" \
  --docker-email=any@valid.email
```

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
  contexts:
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

## Configurations

The `configurations` section of a context object might contain additional configuration data. Currently, only the 
following use case is supported but additional will follow:

- authorization data for helm chart repositories ([see](../deployer/helm.md#access-to-helm-chart-repo-with-authentication))
