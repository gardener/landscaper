package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/gardener/landscaper/apis/core"
)

const (
	Version        = "v1alpha1"
	DeployItemKind = "DeployItem"
)

var DeployItemGVK = schema.GroupVersionKind{
	Group:   core.GroupName,
	Version: Version,
	Kind:    DeployItemKind,
}

func EmptyDeployItemMetadata() *metav1.PartialObjectMetadata {
	metadata := &metav1.PartialObjectMetadata{}
	metadata.SetGroupVersionKind(DeployItemGVK)
	return metadata
}