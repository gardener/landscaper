// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

/*
Package support provides a standard implementation for the object type set
required to implement the OCM repository interface.

This implementation is based on three interfaces that have to implemented:

  - BlobContainer
    is used to provide access to blob data
  - ComponentVersionContainer
    is used to provide access to component version for  component.

The function NewComponentVersionAccessImpl can be used to create an
object implementing the complete ComponentVersionAccess contract.
*/
package support
