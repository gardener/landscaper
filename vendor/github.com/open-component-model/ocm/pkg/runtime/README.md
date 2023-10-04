# Serialization and deserialization of formally typed objects

This package provides support for the de-/serialization of objects into/from a JSON
or YAML representation. 

Objects conforming to this model are called *typed objects*. They have a formal type,
which determines the deserialization method. To be able to infer the type from the
serialization format, it always contains a formal field `type`. So, the minimal
external format of such an object would be (in JSON):

```
{ "type": "my-special-object-type" }
```

A dedicated realm of using typed objects is typically given by a common interface,
the *realm interface*, which all implementations of objects in this realm must 
implement. For example, *Access Specifications* of the OCM all implement the
`AccessSpec` interface. The various implementations of an access specification 
use dedicated objects to represent the property set for a dedicated kind
of specification (for example ` localBlob` or `ociArtifact`), which are able to
provide an access method implementation for the given property set. All those 
specification types together build the access specification realm, with different
specification types implemented as typed objects.

## Core model types

### Simple typed objects

Simple typed objects just feature an unstructured type name. Every type 
directly represents another flavor of the realm interface.

#### Go Types
*interface `TypedObject`* is the common interface for all kinds of typed objects. It
provides access to the name of the type of the object.

*interface `TypedObjectType[T TypeObject]`* is the common interface all type objects
have to implement. It provides information about the type name and a decode (and
encode)method to deserialize/serialize an external JSON/YAML representation of the
object. Its task is to handle the deserialization of instances/objects based of the
type name taken from the deserialization format. 

This interface is parameterized with the Go interface type representing the realm
interface. As a result the `Decode` method provides a correctly typed typed-object
according to the realm interface, which reflects the fact, that all objects in
a realm have to implement this interface.

*type `ObjectTypedObject`* is the minimal implementation of a general typed object
implementing the `TypedObject` interface. The minimal implementation of a typed
object for a dedicated realm might require more according to the requirements of
the realm interface.

*type `ObjectType` is an object providing the type name information of a typed object.

#### Support Functions

With the method `NewTypedObjectTypeByDecoder[T TypedObject]` a new type object
can be created for the realm interface `T` by using a decoder able to map a byte
representation to a typed object implementing the realm interface.

There might be other special implementations of the type interface.

With the method `NewTypedObject` a new minimal typed object can be created
just featuring the type name.

A Go struct implementing a typed object typically looks like:

```go
type MyTypedObject struct {
    ObjectTypeObject `json:",inline"
        ... other object properties
}
```

### Versioned Types

Sometimes it is useful to support several versions for an external representation,
which should all be handled by single *internal* version (Go type) in the rest of the
coding. This can be supported by a flavor of a typed object, the *versioned typed
object*.

A versioned type is described by a type name, which differentiates between 
a common kind and a format version. Here, there is an *internal* program facing
representation given by a Go struct, which can be serialized into different format
versions described by a version string (for example `v1`). Version name and kind are
separated by a slash (`/`). A type name without a version implies the version `v1`.

#### Go Types 

*interface `VersionedTypedObject`* is the common interface for all kinds of typed objects, which provides versioned type. If provides access to the type (like for
plain `TypedObject`s, and its interpretation as a pair of kind and version).

*interface `VersionedTypedObjectType`* is the common interface all type objects for VersionedTypedObjects have to implement.


*type `ObjectVersionedTypedObject`* is the minimal implementation of a typed object
implementing the `VersionedTypedObject` interface.

*type `VersionedObjectType` is an object providing the type information of a
versioned typed object.

#### Support Functions

With the method `NewVersionedTypedObjectTypeByDecoder[T TypedObject]` a new type object
can be created for the realm interface `T` by using a decoder able to map a byte
representation to a typed object implementing the realm interface.

There might be other special implementation of the type interface.

There are several methods to define type objects for versioned typed objects.

- *Single external representation based on the internal representation.*

  With the method `NewVersionedTypedObjectType[T VersionedTypedObject, I VersionedTypedObject]` a new type object is created for a new kind of
  objects in the realm given by the realm interface `T`. `I` is the struct pointer type
  if the internal Go representation also used to describe the single supported
  external format.
  \
  \
  With the method `NewVersionedTypedObject` a new minimal versioned typed object can
  be created just featuring the type information.
  \
  \
  A Go struct implementing a versioned typed object with just a single representation
  typically looks like:

  ```go
  type MyVersionedTypedObject struct {
      ObjectVersionedTypeObject `json:",inline"
          ... other object properties
  }
  ```
  
- *Internal representation of a versioned typed object with multiple external representations.*
  Here, the internal as well as the external representations are described by dedicated
  Go struct types. The internal version is never serialized directly but converted
  first to a Go object describing the external representation described by the version
  content of the type field.
  
  \
  Such a conversion can be done by the `MarshalJSON` method of the internal version.
  The internal versions MUST be able to serialize themselves, because they might be
  embedded as field content in a larger serializable object.

  \
  A better way than implementing it completely on ts own is to use dedicated type
  objects prepared for this. With the method
  `NewVersionedTypedObjectTypeByConverter[T VersionedTypedObject, I VersionedTypedObject, V TypedObject]` such a type object can be created for
  the realm interface `T`, the internal version `I` and the external representation `V`
  (both, `I` and `V` must be struct pointer types, where `I` must implement the realm
  interface `T` (cannot be expressed by Go generics)) and `V` just the typed object
  interface. A correctly typed converter object must be provided, which converts
  between `I` and `V`)
  
  \`
  For such types a dedicated base object representing the type part can be used to
  define the Go type for the internal version. `InternalVersionedTypedObject[T VersionedTypedObject]` keeps information about the type (kind and version) and
  the available serialization versions (used for the conversion to external versions
  as part of the serialization process). It can be created with
  `NewInternalVersionedTypedObject[T VersionedTypedObject]`. Here, a decoder object
  must be given, which can be a version scheme object provided by
  `NewTypeVersionScheme[T VersionedTypedObject, R VersionedTypedObjectType[T]](kind string, reg VersionedTypeRegistry[T, R])`.
  It can be used to register the various available external representation flavors by
  using the dedicated type objects. An example can be found in [`version_test.go`](version_test.go).

  ```go
    type MyVersionedTypedInternalObject struct {
        InternalVersionedTypedObject[MyInternalType] `json:",inline"
            ... other object properties
    }
  ```

  To complete the self-marshalling feature, the `MarshalJSON` method has to be
  implemented just by forwarding the requests to a runtime package function:

  ```go
    func (a MyVersionedTypedInternalObject) MarshalJSON() ([]byte, error) {
      return runtime.MarshalVersionedTypedObject(&a)
    }
  ```
  If the base object from above is not used the version scheme can be passed as
  additional argument.
  

Internally, the different versions are represented with objects implementing the
`FormatVersion[T VersionedTypedObject]` interface, where `T` is the internal
representation. It provides the decoding/encoding for a dedicated format version. It
can explicitly be created with the method  `NewSimpleVersion[T VersionedTypedObject, I VersionedTypedObject]() FormatVersion[T]` for the single version flavor or
`NewConvertedVersion[T VersionedTypedObject, I VersionedTypedObject, V TypedObject](proto V, converter Converter[I, V])` for multiple flavored representations. Those format versions can be used to compose a type object with
`NewVersionedTypedObjectTypeByVersion`

The runtime package handles both case the same way based such format version objects
using an identity converter for the first case.

### Schemes

*interface `Scheme[T TypedObject, R TypedObjectType[T]`* is a factory for typed
objects of a dedicated realm, which hosts deserialization methods (interface `TypedObjectDecoder`) for dedicated types of typed
objects. Basically it implements this task by hosting a set of appropriate decoders
or type objects. It then delegates the task to the handler responsible for
the typename in question. Hereby, `T` is the realm interface and `R` a dedicated Go
interface for a type object. This interface may require more
methods than a plain `TypedObjectType` by providing an own realm specific
interface including the `TypedObjectType[T TypedObject]` interface.

Such a scheme may host types as well as plain decoders.

A type object implements such a decoder interface. For simple type objects, the
serialization is typically just a JSON/YAML serialization of the underlying Go object
representing the property set of this type. This is the default used by the
scheme object, if the decoder does not implement the encoder interface.

The interface type `TypeScheme[T TypedObject, R TypedObjectType[T]]` can be used for
schemes only using type objects and no plain decoders.
It can be achieved for a dedicated realm by the function ` NewTypeScheme[T TypedObject, R TypedObjectType[T]](base ...TypeScheme[T, R])`. Hereby, T is the
realm interface and R a dedicated Go interface for a type object. It may require more
methods than a plain `TypedObjectType`.

The same interface and constructor can be used for schemes for versioned typed
objects, also. The type parameter in such cases must always be an extension of
the `Versioned...` interface types.