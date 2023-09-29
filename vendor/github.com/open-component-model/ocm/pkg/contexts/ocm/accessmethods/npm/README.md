# `npm` - NPM packages in an NPM registry


### Synopsis
```
type: npm/v1
```

Provided blobs use the following media type: `application/x-tgz`

### Description

This method implements the access of an NPM package from an NPM registry.


### Specification Versions

Supported specification version is `v1`

#### Version `v1`

The type specific specification fields are:

- **`registry`** *string*

  Base URL of the NPM registry.

- **`package`** *string*

  The name of the NPM package.

- **`version`** *string*

  The version name of the NPM package.


