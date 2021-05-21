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

	"github.com/gardener/landscaper/apis/config"

	"github.com/go-logr/logr"

	apiextinstall "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	embedFSCrdRootDir = "crdresources"
	readerBufferSize  = 32
)

//go:embed crdresources/landscaper.gardener.cloud*.yaml
var importedCrdFS embed.FS

// CRDManager contains everything required to initialize or update Landscaper CRDs
type CRDManager struct {
	cfg          config.CrdManagementConfiguration
	client       client.Client
	log          logr.Logger
	crdRawDataFS *embed.FS
}

// NewCrdManager returns a new instance of the CRDManager
func NewCrdManager(log logr.Logger, mgr manager.Manager, lsConfig *config.LandscaperConfiguration) (*CRDManager, error) {
	apiExtensionsScheme := runtime.NewScheme()
	apiextinstall.Install(apiExtensionsScheme)
	kubeClient, err := client.New(mgr.GetConfig(), client.Options{Scheme: apiExtensionsScheme})
	if err != nil {
		return nil, fmt.Errorf("failed to setup client to register CRDs: %w", err)
	}

	if _, err = importedCrdFS.ReadDir(embedFSCrdRootDir); err != nil {
		return nil, fmt.Errorf("failed to read from embedded CRDS filesystem: %w", err)
	}

	return &CRDManager{
		cfg:          lsConfig.CrdManagement,
		client:       kubeClient,
		log:          log,
		crdRawDataFS: &importedCrdFS,
	}, nil
}

// EnsureCRDs installs or updates Landscaper CRDs based on Landscaper's configuration
func (crdmgr *CRDManager) EnsureCRDs(ctx context.Context) error {
	if !*crdmgr.cfg.DeployCustomResourceDefinitions {
		crdmgr.log.V(1).Info("Registering Landscaper CRDs disabled by configuration")
		return nil
	}

	crdList, err := crdmgr.crdsFromDir()
	if err != nil {
		return err
	}

	crdmgr.log.V(1).Info("Registering Landscaper CRDs in cluster")
	for _, crd := range crdList {

		existingCrd := &v1.CustomResourceDefinition{}
		err := crdmgr.client.Get(ctx, client.ObjectKey{Name: crd.Name}, existingCrd)
		if err != nil {
			if apierrors.IsNotFound(err) {
				err := crdmgr.createCrd(ctx, &crd)
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

	err = wait.Poll(1*time.Second, 30*time.Second, func() (bool, error) {
		aggregatedStatus := true

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
					}
				case v1.NamesAccepted:
					if crdCondition.Status == v1.ConditionFalse {
						aggregatedStatus = false
					}
				}
			}
		}
		return aggregatedStatus, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (crdmgr *CRDManager) createCrd(ctx context.Context, crd *v1.CustomResourceDefinition) error {
	return crdmgr.client.Create(ctx, crd)
}

func (crdmgr *CRDManager) updateCrd(ctx context.Context, currentCrd, updatedCrd *v1.CustomResourceDefinition) error {
	if !*crdmgr.cfg.ForceUpdate {
		crdmgr.log.V(1).Info("Force update of Landscaper CRDs disabled by configuration")
		return nil
	}

	updatedCrd.ResourceVersion = currentCrd.ResourceVersion
	updatedCrd.UID = currentCrd.UID
	return crdmgr.client.Patch(ctx, updatedCrd, client.MergeFrom(currentCrd))
}

func (crdmgr *CRDManager) crdsFromDir() ([]v1.CustomResourceDefinition, error) {
	crdList := make([]v1.CustomResourceDefinition, 0)

	files, err := crdmgr.crdRawDataFS.ReadDir(embedFSCrdRootDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		data, err := crdmgr.crdRawDataFS.ReadFile(path.Join(embedFSCrdRootDir, file.Name()))
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
