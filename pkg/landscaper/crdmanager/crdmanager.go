// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package crdmanager

import (
	"embed"

	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/crdmanager/pkg/crdmanager"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	embedFSCrdRootDir = "crdresources"
)

//go:embed crdresources/landscaper.gardener.cloud*.yaml
var importedCrdFS embed.FS

// NewCrdManager returns a new instance of the CRDManager
func NewCrdManager(log logr.Logger, mgr manager.Manager, lsConfig *config.LandscaperConfiguration) (*crdmanager.CRDManager, error) {
	return crdmanager.NewCrdManager(log, mgr, lsConfig.CrdManagement, &importedCrdFS, embedFSCrdRootDir)
}
