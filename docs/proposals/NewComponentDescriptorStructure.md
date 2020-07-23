### New ComponentDescriptor structure

*Goals*:
- flatten dependencies into a list; add `type`-attr
- add a metadata/version attr (+ be backwards-compatible)
- document the component_descriptor
- factor out documentation + bindings to a separate repository
- 2nd step: allow non-github-repository-bound components as toplevel-components (or rather: make all components toplevel-components)


- top-level components: gardener-component, componentdefinition (oci)
- dependencies on top-level components are always references

Resolvable components are the top-level components of the component descriptor.
The component descriptor of resolvable component must be retrievable by their type, name and version.
The type defines the mechanism how to the get the component descriptor whereas the name and version uniquely identifies it.

```yaml
version: v2

component:
- type: <dep type>
  name: <dep name>
  version: <dep version>
  <additional type specific attributes>

  dependencies: # name and version has to be unique
  - type: <dep type image | helm | ComponentDefinition | github | gardener-component >
    name: <dep name>
    version: <dep version>
    <additional type specific attributes>

- type: ComponentDefinition
  name: gardener
  version: 1.2.3
  repository: eu.gcr.io/gardener
  
  dependencies:
  - type: gardener-component
    name: gardener
    version: abc
  - type: image
    ref: true
    name: etcd-main
    version: 3.2.2
    repository: eu.gcr.io/etcd
      
  - type: github
    name: github.com/gardener/gardener
    version: 1.0.0
    repository: github.wdf.sap.corp/gardener/gardener

- type: ComponentDefinition
  name: topology
  version: 1.2.3
  repository: eu.gcr.io/gardener
  
  dependencies:
  - type: image
    name: etcd-main
    version: 3.2.2
    repository: eu.gcr.io/etcd

overwrites:
- ref:
      type: <dep type>
      name: <dep name>
      version: <dep version>
  overwrite:
      version: <new version>
      <additional type specific attributes>
    
      dependencies:
      - ref:
          type: <dep type>
          name: <dep name>
          version: <dep version>
        overwrite:
          version: <new version>
          <additional type specific attributes>
  
```

#### Old v1

```yaml
version: v1

components:
- name: github.com/gardener/gardener
  version: 1.2.3
  
  components: 
  - name: github.com/gardener/mcm
    version: 1.2.3
  
  dependencies:
      images:
        - name: etcd-main
          repository: eu.gcr.io/etcd
          version: 3.2.2
- name: github.com/gardener/mcm
  version: 1.2.3
  
  dependencies:
      images:
        - name: etcd-main
          repository: eu.gcr.io/etcd
          version: 3.2.2

- type: ComponentDefinition
  name: virtual-garden
  version: 1.2.3
  
  dependencies:
    components: []
    images:
      - name: etcd-main
        repository: eu.gcr.io/etcd
        version: 3.2.2
overwrites:
- ref:
    name: github.com/gardener/gardener
    version: 1.2.3
  overwrite:
    dependencies:
      images:
      - name: myimage
        version: latest
 
```