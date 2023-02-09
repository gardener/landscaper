# Schemas for Parameters


You can define the type of an import or export parameter using a JSON schema.
We have already seen this in the example about import parameters. 
The [blueprint of that example](../import-parameters/blueprint/blueprint.yaml) has an import parameter `release` 
which is typed as follows:

```yaml
imports:
  - name: release
    type: data
    schema:
      type: object
      properties:
        name:
          type: string
        namespace:
          type: string
```

If a parameter has a more complex structure, it makes sense to define the schema in a separate file to keep the 
`blueprint.yaml` file clean. You can store a schema in a file in the blueprint directory. 
Recall that the blueprint is not just the `blueprint.yaml` file, but the containing
`blueprint` directory. You can put resources like a schema in this directory (or subdirectories).
In this example, we have stored the schema in [blueprint/schemas/release.json](./blueprint/schemas/release.json).

You can then reference the schema in the imports section of the blueprint as follows:

```yaml
imports:
  - name: release
    type: data
    schema:
      $ref: "blueprint://schemas/release.json"
```

The reference starts with `blueprint://` followed by the path inside the `blueprint` directory.



## Procedure

The procedure is as follows:

1. Add the kubeconfig of your target cluster to your [target.yaml](installation/target.yaml).

2. On the Landscaper resource cluster, create namespace `example` and apply
   the [context.yaml](./installation/context.yaml),
   the [target.yaml](installation/target.yaml), 
   and the [installation.yaml](installation/installation.yaml):

   ```shell
   kubectl create ns example
   kubectl apply -f <path to context.yaml>
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml>
   ```


## References

[Import DataMappings](../../../usage/Installations.md#import-data-mappings)
