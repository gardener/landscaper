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
kind: ComponentDefinition

name: my-def
version: 0.5.0

componentDescriptor: abc # could be also a layer in the OCI/in another file

definitionRefs:
- name: abc
  definitionRef: my-other-def:1.0.0
```

```yaml
type: Installation
spec:
    definitionRef: my-def:0.5.0
```

```yaml
type: DeployItem
executors:
- type: helm
  config:
    repo: {{ .dep.nginx.repo }}
    tag: {{ .dep.nginx.tag }}
```

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
version: v2

components:
- type: image | helm | ComponentDefinition | github
  name: gardener
  version: 1.2.3
  
  dependencies:
  - type: <dep type>
    name: <dep name>
    config: <type specific info>

- type: ComponentDefinition
  name: virtual-garden
  version: 1.2.3
  
  dependencies:
  - type: image
    name: etcd-main
    config: 
      repository: eu.gcr.io/etcd
      version: 3.2.2
  - type: <dep type>
    name: <dep name>
    config: <type specific info>

```

```yaml
version: v1

components:
- name: github.com/gardener/gardener
  version: 1.2.3
  
  dependencies:
  components: []
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
```
