---
title: Import Data Mappings
sidebar_position: 2
---

# Import Data Mappings

There are situations where the values in the `DataObjects` are structured differently than the import
parameters of the blueprint, so that a mapping is necessary. The present example demonstrates such an 
import data mapping. It is part of the `Installation`.

An import data mapping can for example be useful in the following situation. Suppose there are two `Installations`,
and the first one exports `DataObjects` which the second one imports. If then the types of the export and import 
parameters do not match exactly, an import data mapping in the second `Installation` can adapt the data.

One can also use an import data mapping to set a fixed value for an input parameter, so that it is not necessary to
create an extra `DataObjects` for the value.


## The Example Mapping

We use the same blueprint as in the [Import Parameters](../import-parameters) example. Recall that it has a string 
parameter `text`, and another parameter `release` which is an object with two string fields `name` and `namespace`.

On the other hand, let's consider the `DataObjects`. Suppose the release name and namespace are given in two separate
`DataObjects` [dataobject-name.yaml](./installation/dataobject-name.yaml) and 
[dataobject-namespace.yaml](./installation/dataobject-namespace.yaml).
In the `imports` section of the [installation](./installation/installation.yaml), we load them into variables
`temp-name` and `temp-namespace`:

```yaml
imports:
  data:
    - name: temp-name                 # temporary variable
      dataRef: my-release-name        # DataObject
    - name: temp-namespace            # temporary variable
      dataRef: my-release-namespace   # DataObject
```

Now we use these variables in the import data mapping. The import data mapping is a template that creates
the data that the blueprint expects, i.e. a `release` object and a `text` string. We build the `release` object from the 
two variables, and we set the `text` parameter to the constant value `hello`:

```yaml
importDataMappings:
  release:                            # import parameter of the blueprint
    name: (( temp-name ))
    namespace: (( temp-namespace ))
  text: hello
```

Note that you must use [spiff](https://github.com/mandelsoft/spiff), rather than GoTemplate as templating language in 
the import data mapping. The reason is that the import data mapping belongs to the yaml manifest of the `Installation`, 
and a GoTemplate would in general not be well-formed yaml.

For more details, see [Import DataMappings](../../../usage/Installations.md#import-data-mappings)


## The Default Mapping

It is possible to skip the import data mapping for some or all import parameters of a blueprint, if no mapping is 
necessary, i.e. if the `DataObjects` contain the data already in the format that the blueprint expects.
Actually, this is what happened in the [Import Parameters](../import-parameters) example.
There we have loaded the `DataObjects` into variables `release` and `text`, whose names and structures matched already
with the parameters of the blueprint: 

```yaml
imports:
  data:
    - name: release
      dataRef: ...
    - name: text
      dataRef: ...
```

As there was no import data mapping defined, by default the variables were mapped unchanged to the corresponding 
import parameters of the blueprint, i.e. they are mapped as if there would have been the following trivial
import data mapping:

```yaml
importDataMappings:
  release: (( release))
  text: (( text ))
```


## Procedure

The procedure is as follows:

1. Add the kubeconfig of your target cluster to your [target.yaml](installation/target.yaml).

2. On the Landscaper resource cluster, create namespace `example` and apply
   the [context.yaml](./installation/context.yaml),
   the [dataobject-name.yaml](./installation/dataobject-name.yaml),
   the [dataobject-namespace.yaml](./installation/dataobject-namespace.yaml),
   the [target.yaml](installation/target.yaml), and the [installation.yaml](installation/installation.yaml):

   ```shell
   kubectl create ns example
   kubectl apply -f <path to context.yaml>
   kubectl apply -f <path to dataobject-name.yaml>
   kubectl apply -f <path to dataobject-namespace.yaml>
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml>
   ```

3. To try out the echo server, first define a port forwarding on the target cluster:

   ```shell
   kubectl port-forward -n example service/echo 8080:80
   ```

   Then open `localhost:8080` in a browser.

   The response should be "hello", which is the text defined
   in the import data mapping in the [installation.yaml](./installation/installation.yaml).


## References

[Import DataMappings](../../../usage/Installations.md#import-data-mappings)

[spiff](https://github.com/mandelsoft/spiff)
