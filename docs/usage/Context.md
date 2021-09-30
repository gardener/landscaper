# Context

The context is a configuration resource containing shared configuration for installations.
This config can contain the repository context, registry pull secrets, or even deployer specific context.

A context can be referenced by installations in the same namespace.

## Basic structure

A context object contains the repository context and an optional list of registry pull secrets.
These registry pull secrets are references to secrets in the same namespace as the context that contains oci registry access credentials.
These credentials can be used to access component descriptors, blueprints or even deployable artifacts like helm charts or oci images.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Context
metadata:
  name: default
  namespace: default

repositoryContext:
  type: ociRegistry
  baseUrl: "example.com"

registryPullSecrets: # additional pull secrets to access component descriptors and blueprints
- name: my-pullsecret
```

## Default Context

Just like kubernetes creates a default service account in every namespace, a default context is created in every namespace.
The default context can be configured in the landscaper configuration `.controllers.context.config.default` as described in the [example](../../examples/00-Landscaper-Configuration.yaml).

The default context object can also be manually modified as the landscaper controller only reconciles configured defaults.

## Configuration

The context controller can be configured in the landscaper config `.controllers.context`.

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
