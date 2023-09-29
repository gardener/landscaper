// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package resourcetypehandlers provides handler for specific resource types. These handlers are responsible for
// accessing the content of resources of the respective type. This may include caching as well as several other
// pre- and postprocessing steps necessary to return the downloaded content in a suitable abstract data type.
//
// In contrast to the corresponding cnudie package, this package does not provide a specific handler for resources of
// type helm but instead provides a separate implementation of the resource interface specifically dedicated for helm
// charts.
// This is due to the fact that the landscaper accesses helm charts through an oci reference or helm repository
// reference directly without having a component descriptor or rather a component version describing these information.
// As a resource, as implemented in the ocm-lib, is always tied to a component version (examine the ocm.ResourceAccess
// struct type for a deeper understanding), a separate resource type was needed.
//
// After implementing a new handler, DO NOT forget to import it in init.go
package resourcetypehandlers
