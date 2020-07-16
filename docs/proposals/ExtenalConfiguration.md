# Providing external configuration

*Assumption*:

We assume that the stakeholder has one or more root components e.g. community setup.
These root components do have a definition (accessible in the oci registry) where all imports are defined.
If mappings in the component are only needed if the key needs to be changed otherwise the default key from the definitions is used.


*Goals*:

- Split external configuration into semantically grouped chunks
  - not one file per key
  - not one big file
- no additional tooling
- chunks should be the same chunks in cluster (no magic processing during upload)

*Agreement*:

We agreed to implement Options 3 ([Add data to landscape config](#add-data-to-landscape-config)).
It has satisfies most of our goals and gives a lot of freedom for easy community deployments as well as strictly versioned cooporate deployments. 

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

### Own config executor


```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: ls-config
spec:
  definition:
    kind: Definition
    exports:
    - key: namespace
      type: string
    executors:
    - name: config
      type: config
      config:
        my:
          data: true

  exports:
  - from: my.data
    to: namespace
```
*Advantages*:
- no new elements
- can be used with the defualt installations import scheduling

*Disadvantages*:
- redefine exports
- need new special executor

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

  exports:
  - from: root.namespace
    to: namespace

  staticData:
  - value:
      root:
        namespace: default
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
- need specific handling for secrets (vault integration, etc..)

---

Also make it possible to specify staticData in defintions.
With that it would be possible to have completely versioned component with versioned configuration.

:warning: secrets in oci registry
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: community-setup
spec:
  definitionRef: sap-dev-ls:0.120.0
```

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Definition
metadata:
  name: sap-dev-ls
spec:
  definitions:
  - ref: community-setup:0.5.0
  - ref: config:0.2.0
```

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Definition
metadata:
  name: config
spec:
  exports:
  - key: namespace
    type: string

  staticData:
  - value:
      namespace: default
  - fromFile:
      path: mybloc/namesapce
      jsonpath: abc
```