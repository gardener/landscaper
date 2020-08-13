# Component Descriptor

The ComponentDescriptor is the BOM(Bill of Materials) of a Component(Definition?) that contains all dependencies and their versions.

Dependencies can be of a specific type/kind whereas all kinds of dependencies should be supportable.
By now the planned types are `Images`, `OCI Helm Charts` and `ComponentDefinitions`.

The ComponentDescriptor also contains overwrites that can be used to overwrite the location (maybe also version) of a dependency.
To support this feature, all dependencies must have a unique name which is used to reference the type specific dependency information.

*Goals*:
- Dev dependencies -> overwrite versions of dependencies 
  - Problem: may need also a dev version of the parent (mappings in the parents definition) or the actual installation mappings
  - Worst case: need own dev version of parent
- Dependencies in restricted environments
  - The Landscaper should be able to consume dependencies from another OCI registry.
- Dependencies should be somehow specified in the imports/ComponentDefinition?
- Deployers/DeployItems must have access to the dependencies and their (maybe) overwritten locations/info.
- Component specific dependencies

*Assumptions*:

```yaml
type: Installation
spec:
    blueprintRef:
      baseUrl: eu.gcr.io/...
      component/ref: my-other-def
      kind: localResource
      resource: blueprint
```

```yaml
kind: Blueprint

name: my-def
version: 0.5.0

import:
- key: etcd
  component:
    ref: github.com/gardener/mcm
    kind: externalResource
    resource: etcd
    type: ociImage


blueprintRefs:
- name: abc
  blueprintRef:
    component/ref: my-other-def
    kind: localResource
    resource: blueprint
- name: cde
  blueprintRef:
    component/ref: my-other-def
    kind: localResource
    resource: blueprint
```

```yaml
type: DeployItem
executors:
- type: helm
  config:
    image: {{ imports.etcd.access.imageReference }}
    image2: {{ compdesc."github.com/gardener/mcm".externalResources.etcd.access.ImageReference }}
```

- the root installation specifies the component from where the component descriptor is fetched and the blueprint resource in this component desc.
  - make base url in the installation configurable
- aggregated blueprints specify only the blueprint's component and blueprints location (kind: local/external and name)
  - subinstallation blueprint's components have to be defined as componentReference in the aggregated component descriptor.
  - the baseUrl is automatically given by the repository Context of the aggregated component
  - this basUrl is then also propagated to the sub installations
  
- local or external resources of a component can be accessed via import declartion with referece/componentName, kind and resourceName.
- the landscaper also translated the components, the local and external resources in to a map so that one can access it by index (no need to for looping over the resources array).

<hr>

Old component descriptor
- ComponentDefinitions and their Refs have to state the unique identifier of the Definition not the actual ref. `e.g. "gardener:1.0.0" not "eu.gcr.io/my-proj/gardener:1.0.0"`
- Installations may contain the real ref.

**Ideas**:

_Separate File_:<br>
Pro:
- Use already existing and needed ComponentDescriptor File
- Have dependencies separated from runtime values.
- Better handling in component as a common extensible descriptor is used to describe dependecies (we need to have that at least for private cloud deployments).
Con:
- need to maintain an additional object in the registry
- new first class citizen entity in the landscaper

_As Import_:<br>
Pro:
- use existing import validation
Con:
- additional maintenance of dependencies in the ComponentDescriptor and the ComponentDefinition
- whole lot of new imports that need to be maintained and mapped in aggregations
- translation of ComponentDescriptor to imports (how can be do the mapping?)


### New ComponentDescriptor structure

- flatten dependencies into a list; add `type`-attr
- add a metadata/version attr (+ be backwards-compatible)
- document the component_descriptor
- factor out documentation + bindings to a separate repository
- 2nd step: allow non-github-repository-bound components as toplevel-components (or rather: make all components toplevel-components)

```yaml
meta:
  schemaVersion: 'v2'

component:
  name: 'github.com/gardener/gardener'
  version: 'v1.7.2'

  provider: internal
  repositoryContexts:
  - type: 'ociRegistry'
    baseUrl: 'eu.gcr.io/gardener-project/dev' # => eu.gcr.io/gardener-project/dev/github.com/gardener/gardener:v1.7.2
  sources: []
  componentReferences:
  - ...
  localResources: 
  - name: blueprint
    type: blueprint
    access:
      type: ociRegistry
      reference: eu.gcr.io/gardener-project/blueprints/gardener:v1.7.2

  externalResources:
  - name: 'hyperkube'
    version: 'v1.16.4'
    type: 'ociImage'
    access:
      type: 'ociRegistry'
      # image_reference attribute is implied by `ociImage` type
      imageReference: 'eu.gcr.io/gardener-project/gardener/apiserver:v1.7.2'

```

```yaml
version: v1

components:
- name: github.com/gardener/gardener
  version: 1.2.3
  
  components: []
  
  dependencies:
      images:
        - type: image
          name: etcd-main
          config: 
            repository: eu.gcr.io/etcd
            version: 3.2.2

- type: ComponentDefinition
  name: virtual-garden
  version: 1.2.3
  
  dependencies:
    components: []
    images:
      - type: image
        name: etcd-main
        config: 
          repository: eu.gcr.io/etcd
          version: 3.2.2
overwrites:
- name: github.com/gardener/gardener
  version: 1.2.3
  
  
```


````yaml

# github.com/gardener/mcm / external resources / etcd => name


references:
- name: github.com/gardener/mcm # basepath + name + version => component descriptor
  version: 1.5.0


externalResource:
- name: hyperkube
  version: 1.16.4 => 1.16.x
  access:
    imageRef: abc
- name: hyperkube
  version: 1.17.2
```
