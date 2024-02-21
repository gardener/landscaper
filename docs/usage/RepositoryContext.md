---
title: Repository Context
sidebar_position: 4
---

# Repository Context

## Definition

A repository context describes the access to a repository that is used to resolve component version references.
It features the following fields:

- **`type`** *string*
  
  The type of the repository, e.g. `ociRegistry`.

- *type-specific fields*

  Additional fields describe attributes to address the dedicated repository instance according to the type.

A repository context is used all over the landscaper objects to resolve component descriptor references. It can be specified either directly in [Installations](./Installations.md#component-descriptor) or in [Context](./Context.md) objects referenced by Installations.

A repository context is typically described by a field `repositoryContext` in objects referring to a such a context.

## Supported Types

### OCI Registries

A component repository based on an OCI registry is described by the following additional fields:

- **`baseUrl`** *host URL*

  Scheme, host, and port of the underlying OCI registry.

- **`subPath`** *string* (optional)

  The repository prefix for mapping component names to OCI artifact repositories.

- **`componentNameMapping`** *string* (optional)

  To support OCI registries with a limitation for repository names, there are several modes to map component names to OCI artifact repositories:

  - **`urlPath`** (default)

    Join `subPath` and component name.
  
  - **`sha256-digest`**

    Encode the component name with a sha256 digest, appended to the `subPath`.
