# Providing external configuration 

*Assumption*:

We assume that the stakeholder has one or more root components e.g. community setup.
These root components do have a definition (accessible in the oci registry) where all imports are defined.
If mappings in the component are only needed if the key needs to be changed otherwise the default key from the definitions is used.


*Goals*:

- Logically split external configuration into chunks
  - not one file per key
  - not one big file
- no additional tooling
- chunks should be the same chunks in cluster (no magic processing during upload)


## Options

### Make it part of the imports

Make a import directly configurable withing their import mapping.
Instead of mapping values `from-to` the values could be directly set.
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: community-setup
spec:
  imports:
  - to: namespace
    value: default
```

*Advantages*:
- no additional resources
- no magic

*Disadvantages*:
- big installation
- unreadable root installation
  - bad maintenance

### Extra Deployer
Extra deployers creates installation on the fly

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: ls-config
spec:  
  definitionRef: abc
  executions:
  - type: config
    config:
      secrets:
      - name: main
      - name: main-secrets
```

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: generated-ls
spec: 
  definition: 
    mydef: abc
  exports:
  - key: root.namespace
    type: string
```

*Advantages*:
- real installations
- no dynamic exports for users
- multiple secrets for logical splitting

*Disadvantages*:
- magically generated installation
- dynamic definition
- dynamic installations
- extra jsonpath definition for exported keys othwerwise read from root components with a lot of custom logic
- new deployer


### Add data to landscape config

Configuration can be defined in separate secrets and deployed to cluster.
These secrets are then reference in the root installation(s) as staticData which 
is then also used for satisfying imports.

All secrets are combined and data is searched based on keys as jsonpath

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: community-setup
spec:
  definitionRef: community-setup-def:1.0.0
  imports: # defines optional mapping for better structuring in the config
  - from: root.namespace
    to: namespace
    
  staticData:
  - name: main
  - name: main-secrets
```

*Advantages*:
- no additional resources
- logical splitting is possible
- easy local testing

*Disadvantages*:
- how to track changes?
- magic happens when import key is jsonpath.
