// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	lsversion "github.com/gardener/landscaper/pkg/version"
)

// DefaultConfiguration sets the defaults for the container deployer configuration.
func DefaultConfiguration(obj *containerv1alpha1.Configuration) {
	containerv1alpha1.SetDefaults_Configuration(obj)
	if len(obj.InitContainer.Image) == 0 {
		version := lsversion.Get().GitVersion
		// default lsversion to latest if the gitversion is 0.0.0-dev
		if version == "0.0.0-dev" {
			version = "latest"
		}
		obj.InitContainer.Image = fmt.Sprintf("europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper/github.com/gardener/landscaper/container-deployer/images/container-deployer-init:%s", version)
	}
	if len(obj.WaitContainer.Image) == 0 {
		version := lsversion.Get().GitVersion
		// default lsversion to latest if the gitversion is 0.0.0-dev
		if version == "0.0.0-dev" {
			version = "latest"
		}
		obj.WaitContainer.Image = fmt.Sprintf("europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper/github.com/gardener/landscaper/container-deployer/images/container-deployer-wait:%s", version)
	}
}

// DecodeProviderStatus decodes a RawExtension to a container status.
func DecodeProviderStatus(raw *runtime.RawExtension) (*containerv1alpha1.ProviderStatus, error) {
	status := &containerv1alpha1.ProviderStatus{}
	if raw != nil {
		if _, _, err := serializer.NewCodecFactory(api.LandscaperScheme).UniversalDecoder().Decode(raw.Raw, nil, status); err != nil {
			return nil, err
		}
	}
	return status, nil
}
