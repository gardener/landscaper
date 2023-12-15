package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/gardener/landscaper/apis/core"
)

const (
	Version          = "v1alpha1"
	DeployItemKind   = "DeployItem"
	ExecutionKind    = "Execution"
	InstallationKind = "Installation"
)

var DeployItemGVK = schema.GroupVersionKind{
	Group:   core.GroupName,
	Version: Version,
	Kind:    DeployItemKind,
}

var ExecutionGVK = schema.GroupVersionKind{
	Group:   core.GroupName,
	Version: Version,
	Kind:    ExecutionKind,
}

var InstallationGVK = schema.GroupVersionKind{
	Group:   core.GroupName,
	Version: Version,
	Kind:    InstallationKind,
}

var podGVK = schema.GroupVersionKind{
	Group:   "",
	Version: "v1",
	Kind:    "Pod",
}

func EmptyDeployItemMetadata() *metav1.PartialObjectMetadata {
	metadata := &metav1.PartialObjectMetadata{}
	metadata.SetGroupVersionKind(DeployItemGVK)
	return metadata
}

func EmptyExecutionMetadata() *metav1.PartialObjectMetadata {
	metadata := &metav1.PartialObjectMetadata{}
	metadata.SetGroupVersionKind(ExecutionGVK)
	return metadata
}

func EmptyInstallationMetadata() *metav1.PartialObjectMetadata {
	metadata := &metav1.PartialObjectMetadata{}
	metadata.SetGroupVersionKind(InstallationGVK)
	return metadata
}

func EmptyPodMetadata() *metav1.PartialObjectMetadata {
	metadata := &metav1.PartialObjectMetadata{}
	metadata.SetGroupVersionKind(podGVK)
	return metadata
}

func EmptyPodMetadataList() *metav1.PartialObjectMetadataList {
	metadata := &metav1.PartialObjectMetadataList{}
	metadata.SetGroupVersionKind(podGVK)
	return metadata
}
