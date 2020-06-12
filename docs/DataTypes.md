# DataTypes

DataTypes are the base components of the landscaper.
DataTypes describe the data that is im- and exported by the components (installations).

DataTypes are a immutable kubernetes resource that is installed in the cluster among the installations.
As DataTypes are immutable you have to be careful when creating one as currently no migration between datatypes is possible.

### Define a new DataType

DataTypes are described with the following structure whereas datatypes definition is basically a wrapper around a OpenapiV3 schema definition.


```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DataType
scheme:
  openAPIV3Schema: {} # here goes the openapiv3spec
```

For detailed documentation see https://swagger.io/docs/specification/data-models/

### :warning: Differences to the APIV3Spec
- floats have to be defined as strings as the kubernetes api does not allow float numbers.
  E.g. 1.234 -> "1.234"
- other types can be referenced via the `$ref` field but without a leading `#/`.<br>
  This means that another datatype can be simply referenced as `$ref: my-custom-type`

### Examples:
Additional Examples can be found in the example folder `/examples`

``` yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DataType
metadata:
  name: image
scheme:
  openAPIV3Schema:
    type: object
    properties:
      name:
        type: string
      repository:
        type: string
      tag:
        type: string
```

``` yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DataType
metadata:
  name: url
scheme:
  openAPIV3Schema:
    type: string
    format: uri
```

``` yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DataType
metadata:
  name: my-referenced-type
scheme:
  openAPIV3Schema:
    type: string
    format: uri
---
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DataType
metadata:
  name: my-custom-type
scheme:
  openAPIV3Schema:
    type: object
    properties: 
      name:
        type: string
      url:
        $ref: my-referenced-type
```

