// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package crdmanager

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"path"
	"time"

	apiextinstall "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

const (
	readerBufferSize = 1024
)

// CRDManager contains everything required to initialize or update CRDs
type CRDManager struct {
	cfg          config.CrdManagementConfiguration
	client       client.Client
	crdRawDataFS *embed.FS
	crdRootDir   string
}

// NewCrdManager returns a new instance of the CRDManager
func NewCrdManager(mgr manager.Manager, config config.CrdManagementConfiguration, crdRawDataFS *embed.FS, crdRootDir string) (*CRDManager, error) {
	apiExtensionsScheme := runtime.NewScheme()
	apiextinstall.Install(apiExtensionsScheme)
	kubeClient, err := client.New(mgr.GetConfig(), client.Options{Scheme: apiExtensionsScheme})
	if err != nil {
		return nil, fmt.Errorf("failed to setup client to register CRDs: %w", err)
	}

	if _, err = crdRawDataFS.ReadDir(crdRootDir); err != nil {
		return nil, fmt.Errorf("failed to read from embedded CRDS filesystem: %w", err)
	}

	return &CRDManager{
		cfg:          config,
		client:       kubeClient,
		crdRawDataFS: crdRawDataFS,
		crdRootDir:   crdRootDir,
	}, nil
}

// EnsureCRDs installs or updates CRDs based on the configuration
func (crdmgr *CRDManager) EnsureCRDs(ctx context.Context) error {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "EnsureCRDs"})

	if !*crdmgr.cfg.DeployCustomResourceDefinitions {
		logger.Info("Registering CRDs disabled by configuration")
		return nil
	}

	crdList, err := crdmgr.crdsFromDir()
	if err != nil {
		return err
	}

	logger.Info("Registering CRDs in cluster")
	for _, crd := range crdList {

		existingCrd := &v1.CustomResourceDefinition{}
		err := crdmgr.client.Get(ctx, client.ObjectKey{Name: crd.Name}, existingCrd)
		if err != nil {
			if apierrors.IsNotFound(err) {
				err := crdmgr.client.Create(ctx, &crd)
				if err != nil {
					return err
				}
				continue
			}
			return err
		}

		err = crdmgr.updateCrd(ctx, existingCrd, &crd)
		if err != nil {
			return err
		}
	}

	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
		aggregatedStatus := true
		causingCRDs := sets.NewString()

		for _, crd := range crdList {
			if !aggregatedStatus {
				return aggregatedStatus, nil
			}
			crdResult := &v1.CustomResourceDefinition{}
			err := crdmgr.client.Get(ctx, client.ObjectKey{Name: crd.Name}, crdResult)
			if err != nil {
				return false, err
			}

			for _, crdCondition := range crdResult.Status.Conditions {
				switch crdCondition.Type {
				case v1.Established:
					if crdCondition.Status != v1.ConditionTrue {
						aggregatedStatus = false
						causingCRDs.Insert(crd.Name)
					}
				case v1.NamesAccepted:
					if crdCondition.Status == v1.ConditionFalse {
						aggregatedStatus = false
						causingCRDs.Insert(crd.Name)
					}
				}
			}
		}
		logger.Debug("Not all CRDs are ready", "unreadyCRDs", causingCRDs.List())
		return aggregatedStatus, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (crdmgr *CRDManager) updateCrd(ctx context.Context, currentCrd, updatedCrd *v1.CustomResourceDefinition) error {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "updateCrd"})

	if !*crdmgr.cfg.ForceUpdate {
		logger.Info("Force update of CRDs disabled by configuration")
		return nil
	}

	updatedCrd.ResourceVersion = currentCrd.ResourceVersion
	updatedCrd.UID = currentCrd.UID
	return crdmgr.client.Patch(ctx, updatedCrd, client.MergeFrom(currentCrd))
}

func (crdmgr *CRDManager) crdsFromDir() ([]v1.CustomResourceDefinition, error) {
	crdList := make([]v1.CustomResourceDefinition, 0)

	files, err := crdmgr.crdRawDataFS.ReadDir(crdmgr.crdRootDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		data, err := crdmgr.crdRawDataFS.ReadFile(path.Join(crdmgr.crdRootDir, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read CRD file %q: %w", file.Name(), err)
		}

		decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), readerBufferSize)
		crd := &v1.CustomResourceDefinition{}
		err = decoder.Decode(crd)
		if err != nil {
			return nil, fmt.Errorf("failed to decode CRD from YAML file %q: %w", file.Name(), err)
		}

		crdList = append(crdList, *crd)
	}

	return crdList, nil
}
