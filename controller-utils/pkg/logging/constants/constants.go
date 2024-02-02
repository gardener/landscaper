// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package constants

const (
	// KeyReconciledResource is for 'namespace/name' of the resource which is currently being reconciled.
	// Do not use this field directly, it is added automatically by logging.StartReconcileFromContext or log.StartReconcile.
	KeyReconciledResource = "reconciledResource"
	// KeyReconciledResourceKind is for the kind of the resource which is being reconciled by this controller.
	// Do not use this field directly, it is added by log.Reconciles.
	KeyReconciledResourceKind = "reconciledResourceKind"
	// KeyOperation is for the current operation
	KeyOperation = "operation"
	// KeyMethod is for the currently executed method
	KeyMethod = "method"
	// KeyResource is for 'namespace/name' of a resource.
	// For referencing the resource which is currently being reconciled, use KeyReconciledResource instead.
	KeyResource = "resource"
	// KeyResourceNonNamespaced is for the name of a non-namespaced resource.
	KeyResourceNonNamespaced = "resourceNonNamespaced"
	// KeyResourceKind is for the kind of the referenced resource. Meant to be used in combination with KeyResource.
	// For the kind of the resource which is currently being reconciled, use KeyReconciledResourceKind instead.
	KeyResourceKind = "resourceKind"
	// KeyResourceGroup is for the group of the referenced resource. Meant to be used in combination with KeyResource.
	KeyResourceGroup = "resourceGroup"
	// KeyResourceVersion is for the version of the referenced resource. Meant to be used in combination with KeyResource.
	KeyResourceVersion = "resourceVersion"
	// KeyInstallationPhase is for the phase of an installation.
	KeyInstallationPhase = "installationPhase"
	// KeyExecutionPhase is for the phase of an execution.
	KeyExecutionPhase = "executionPhase"
	// KeyDeployItemPhase is for the phase of a deployitem.
	KeyDeployItemPhase = "deployitemPhase"
	// KeyReadID is for a reader ID.
	KeyReadID = "readID"
	// KeyWriteID is for a writer ID.
	KeyWriteID = "writeID"
	// KeyGeneration is for the generation of a resource.
	KeyGeneration = "generation"
	// KeyObservedGeneration is for the observed generation of a resource.
	KeyObservedGeneration = "observedGeneration"
	// KeyGenerationOld is for the old generation of a resource.
	KeyGenerationOld = "oldGeneration"
	// KeyGenerationNew is for the new generation of a resource.
	KeyGenerationNew = "newGeneration"
	// KeyResourceVersionOld is for the old resource version of a resource.
	KeyResourceVersionOld = "oldResourceVersion"
	// KeyResourceVersionNew is for the new resource version of a resource.
	KeyResourceVersionNew = "newResourceVersion"
	// KeyJobID is for the ID of the current job.
	KeyJobID = "jobID"
	// KeyJobIDFinished is for the ID of the finished job.
	KeyJobIDFinished = "jobIDFinished"
	// KeyCDName is the name of a component descriptor.
	KeyCDName = "cdName"
	// KeyVersion is for referencing a version
	KeyVersion = "version"
	// KeyDeletionTimestamp is for referencing a deletion timestamp. The value should be of type time.Time.
	KeyDeletionTimestamp = "deletionTimestamp"
	// KeyIndex is for generic numeric indexes.
	KeyIndex = "index"
	// KeyFileName is for the name of a file.
	KeyFileName = "fileName"
	// KeyServiceAccount is for a kubernetes service account.
	KeyServiceAccount = "serviceAccount"
	// KeyDeployItemType is for the type of deployitem which is handled by the controller.
	KeyDeployItemType = "deployItemType"
	// KeyError is for logging an error message without using 'logger.Error' for whatever reason.
	KeyError = "error"
	// KeyGroupVersionKind is for listing group, version, and kind of a resource.
	KeyGroupVersionKind = "groupVersionKind"
	// KeyStatus is for a status of whatever kind.
	KeyStatus = "status"
	// KeyOperationAnnotation is for logging the value of the current operation annotation.
	KeyOperationAnnotation = "operationAnnotation"
	// KeyReason is for providing a reason.
	KeyReason = "reason"
	// KeyPhase is for a generic phase. For the phases of Installations, Executions, or DeployItems, use the more specific constants.
	KeyPhase = "phase"
	// KeyExportKey is for the key that the value from JSONPath is exported to.
	KeyExportKey = "exportKey"
	// KeyManagedResourcePolicy is for the manage policy of a resource.
	KeyManagedResourcePolicy = "managedResourcePolicy"
	// KeyString is for generic strings.
	KeyString = "someString"

	// MsgStartReconcile is the message which is displayed at the beginning of a new reconcile loop.
	MsgStartReconcile = "Starting reconcile"
	// MsgStartMethod is a message for logging the beginning of a method
	MsgStartMethod = "Starting method"
)
