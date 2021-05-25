# Conditional Imports

The basics of import definitions in blueprints are explained [here](./Blueprints.md). This document focuses on an advanced feature for import definitions: conditional imports.

The idea is simple: sometimes some values are only needed if some others - which are optional - are also provided. You can also think about this the other way around, for example: if no url for a backup service is provided, we don't need any password for accessing it either. 

Modelling this in a blueprint's import definitions is difficult - one can either mark the password as optional, but then landscaper can't enforce its existence, even if the url import is satisfied. Or they are marked as required, but then they need to be provided, even if they are not used.

The concept of 'conditional imports' aims to solve this problem. The above example could look like this in a blueprint's imports definition:

```yaml
imports:
- name: backup-storage-url
  required: false
  schema:
    type: string
  imports:
  - name: backup-storage-password
    schema:
      type: string
```

Put in simple words, this definitions says "if the 'backup-storage-url' import is satisfied, the 'backup-storage-password' import has to be too".
Please note that the `backup-storage-url` import is marked as optional. This always has to be the case and is validated by the landscaper.

Nested 'conditional import' definitions follow exactly the same structure as top-level import definitions, which means they can also be marked as optional and contain further conditional imports depending on them.

It is currently not possible to construct a condition that checks an existing import for a specific value - only whether an import is satisfied can be used for the condition.

Unlike imports, exports can neither be made optional nor conditional.


## Conditional Imports and Subinstallation Templates

Sometimes a subinstallation has an import that comes from an optional or conditional import of the parent installation. This means that the subinstallation template in the parent's blueprint will have imports listed which might not be there during runtime.

See the below snippet of a blueprint as an example:
```yaml
... # redacted
imports:
- name: foo
  required: false
  schema:
    type: string
  imports:
  - name: bar
    schema:
      type: string

subinstallations:
- apiVersion: landscaper.gardener.cloud/v1alpha1
  kind: InstallationTemplate
  name: subinst-import
  imports:
    data:
    - name: internalFoo
      dataRef: foo
    - name: internalBar
      dataRef: bar
  blueprint:
    filesystem:
      blueprint.yaml: |
        ... # redacted
        imports:
        - name: internalFoo
          required: false
          schema:
            type: string
        - name: internalBar
          required: false
          schema:
            type: string
```
The parent blueprint defines two imports, `foo` and `bar`, with `foo` being optional and `bar` being conditional (depending on `foo`). The subinstallation blueprint defines two imports `internalFoo` and `internalBar`, both being optional. The subinstallation template wires `internalFoo` to the parent's `foo` import and `internalBar` to the parent's `bar` import. 
If both imports of the parent are satisfied, this isn't a problem. However, if the parent's installation does not actually import `foo` and `bar`, then the automatically created subinstallation would not be satisfied, since it would try to import `foo` and `bar` from its parent and the resource would not become ready - despite both imports being optional in the subinstallation's blueprint.

To avoid this problem, **landscaper will remove all imports that are wired to unsatisfied optional or conditional parent imports** when creating the subinstallations.

To better describe the scope of this automatic import removal:
- imports will only be removed if the parent import they are wired to is optional or conditional - imports that are wired to a required parent import will not be removed
- imports will only be removed if they are actually wired to a parent's import - those which are satisfied by a sibling's export will not be removed
- even though imports depending on parent's optional/conditional imports are being removed, the subinstallation will still not succeed if the removed imports are not optional in its own blueprint