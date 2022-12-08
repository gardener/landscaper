// SPDX-FileCopyrightText: 2021 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package core

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TargetSyncList contains a list of TargetSync objects
type TargetSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TargetSync `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// The TargetSync is created targets from secrets.
type TargetSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec contains the specification
	Spec TargetSyncSpec `json:"spec"`

	// Status contains the status
	// +optional
	Status TargetSyncStatus `json:"status"`
}

// TargetSyncSpec contains the specification for a TargetSync.
type TargetSyncSpec struct {
	// SourceNamespace describes the namespace from where the secrets should be synced
	SourceNamespace string `json:"sourceNamespace"`

	// SecretRef references the secret that contains the kubeconfig to the namespace of the secrets to be synced.
	SecretRef LocalSecretReference `json:"secretRef"`

	// CreateTargetToSource specifies if set on true, that also a target is created, which references the secret in SecretRef
	// +optional
	CreateTargetToSource bool `json:"createTargetToSource,omitempty"`

	// TargetToSourceName is the name of the target referencing the secret defined in SecretRef if CreateTargetToSource
	// is set on true. If TargetToSourceName is empty SourceNamespace is used instead.
	// +optional
	TargetToSourceName string `json:"targetToSourceName,omitempty"`

	// SecretNameExpression defines the names of the secrets which should be synced via a regular expression according
	// to https://github.com/google/re2/wiki/Syntax with the extension that * is also a valid expression and matches
	// all names.
	// if not set no secrets are synced
	// +optional
	SecretNameExpression string `json:"secretNameExpression"`

	// ShootNameExpression defines the names of shoot clusters for which targets with short living access data
	// to the shoots are created via a regular expression according to https://github.com/google/re2/wiki/Syntax with
	// the extension that * is also a valid expression and matches all names.
	// if not set no targets for the shoots are created
	// +optional
	ShootNameExpression string `json:"shootNameExpression"`

	// TokenRotation defines the data to perform an automatic rotation of the token to access the source cluster with the
	// secrets to sync. The token expires after 90 days and will be rotated every 60 days.
	// +optional
	TokenRotation *TokenRotation `json:"tokenRotation,omitempty"`
}

type TokenRotation struct {
	// Enabled defines if automatic token is executed
	Enabled bool `json:"enabled,omitempty"`
}

// TargetSyncStatus contains the status of a TargetSync.
type TargetSyncStatus struct {
	// ObservedGeneration is the most recent generation observed.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration"`

	// Last time the status was updated
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// LastErrors describe the last errors
	// +optional
	LastErrors []string `json:"lastErrors,omitempty"`

	// Last time the token was rotated
	// +optional
	LastTokenRotationTime *metav1.Time `json:"lastTokenRotationTime,omitempty"`
}
