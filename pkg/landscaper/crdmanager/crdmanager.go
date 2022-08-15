// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package crdmanager

import (
	"embed"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/controller-utils/pkg/crdmanager"
)

const (
	embedFSCrdRootDir = "crdresources"
)

//go:embed crdresources/landscaper.gardener.cloud*.yaml
var importedCrdFS embed.FS

// NewCrdManager returns a new instance of the CRDManager
func NewCrdManager(mgr manager.Manager, lsConfig *config.LandscaperConfiguration) (*crdmanager.CRDManager, error) {
	return crdmanager.NewCrdManager(mgr, lsConfig.CrdManagement, &importedCrdFS, embedFSCrdRootDir)
}
