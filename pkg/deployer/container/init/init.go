// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package init

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	blueprintsoci "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints/oci"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/credentials"
)

// Run downloads the import config, the component descriptor and the blob content
// to the paths defined by the env vars.
// It also creates all needed directories.
func Run(ctx context.Context, log logr.Logger) error {
	opts := &options{}
	opts.Complete(ctx)
	if err := opts.Validate(); err != nil {
		return err
	}

	restConfig, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return err
	}

	var kubeClient client.Client
	if err := wait.ExponentialBackoff(opts.DefaultBackoff, func() (bool, error) {
		var err error
		kubeClient, err = client.New(restConfig, client.Options{
			Scheme: kubernetes.LandscaperScheme,
		})
		if err != nil {
			log.Error(err, "unable to build kubernetes client")
			return false, nil
		}
		return true, nil
	}); err != nil {
		return err
	}
	deployItem := &lsv1alpha1.DeployItem{}
	if err := kubeClient.Get(ctx, opts.DeployItemKey, deployItem); err != nil {
		return err
	}

	regAcc, err := createRegistryFromDockerAuthConfig(ctx, log, kubeClient, deployItem)
	if err != nil {
		return err
	}

	// create all directories
	log.Info("create directories")
	if err := os.MkdirAll(path.Dir(opts.ExportsFilePath), os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Dir(opts.ComponentDescriptorFilePath), os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(opts.ContentDirPath, os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(opts.StateDirPath, os.ModePerm); err != nil {
		return err
	}
	log.Info("all directories have been successfully created")

	if deployItem.Spec.BlueprintRef != nil {
		log.Info("get component descriptor")
		cd, err := regAcc.ComponentsRegistry().Resolve(ctx, deployItem.Spec.BlueprintRef.RepositoryContext, deployItem.Spec.BlueprintRef.ObjectMeta())
		if err != nil {
			return errors.Wrapf(err, "unable to resolve component descriptor for ref %#v", deployItem.Spec.BlueprintRef)
		}
		cdList, err := cdutils.ResolveEffectiveComponentDescriptorList(ctx, regAcc.ComponentsRegistry(), *cd)
		if err != nil {
			return errors.Wrapf(err, "unable to resolve component descriptor references for ref %#v", deployItem.Spec.BlueprintRef)
		}

		cdListJSONBytes, err := json.Marshal(cdutils.ConvertFromComponentDescriptorList(cdList))
		if err != nil {
			return errors.Wrap(err, "unable to unmarshal mapped component descriptor")
		}
		if err := ioutil.WriteFile(opts.ComponentDescriptorFilePath, cdListJSONBytes, os.ModePerm); err != nil {
			return errors.Wrapf(err, "unable to write mapped component descriptor to file %s", opts.ComponentDescriptorFilePath)
		}
	}

	if deployItem.Spec.BlueprintRef != nil {
		log.Info("get blueprint content")
		log.Info(fmt.Sprintf("fetching blueprint for %#v", deployItem.Spec.BlueprintRef))
		blueprint, err := blueprints.Resolve(ctx, regAcc, *deployItem.Spec.BlueprintRef)
		if err != nil {
			return errors.Wrap(err, "unable to fetch blueprint from registry")
		}

		osFS := afero.NewOsFs()
		if err := copyFS(blueprint.Fs, osFS, "/", opts.ContentDirPath); err != nil {
			return err
		}
		log.Info(fmt.Sprintf("blueprint content successfully downloaded to %s", opts.ContentDirPath))
	}

	log.Info("get state")

	return nil
}

func copyFS(src, dst afero.Fs, srcPath, dstPath string) error {
	return afero.Walk(src, srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		dstFilePath := filepath.Join(dstPath, path)
		if info.IsDir() {
			if err := dst.MkdirAll(dstFilePath, info.Mode()); err != nil {
				return err
			}
			return nil
		}

		file, err := src.OpenFile(path, os.O_RDONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		return afero.WriteReader(dst, dstFilePath, file)
	})
}

// registries is a internal struct that implements the registry accessors interface
type registries struct {
	blueprintsRegistry blueprintsregistry.Registry
	componentsRegistry componentsregistry.Registry
}

var _ lsoperation.RegistriesAccessor = &registries{}

func (r registries) BlueprintsRegistry() blueprintsregistry.Registry {
	return r.blueprintsRegistry
}

func (r registries) ComponentsRegistry() componentsregistry.Registry {
	return r.componentsRegistry
}

// todo: add retries
func createRegistryFromDockerAuthConfig(ctx context.Context, log logr.Logger, kubeClient client.Client, deployItem *lsv1alpha1.DeployItem) (lsoperation.RegistriesAccessor, error) {
	secrets := make([]corev1.Secret, len(deployItem.Spec.RegistryPullSecrets))
	for i, secretRef := range deployItem.Spec.RegistryPullSecrets {
		secret := corev1.Secret{}
		if err := kubeClient.Get(ctx, secretRef.NamespacedName(), &secret); err != nil {
			return nil, err
		}
		secrets[i] = secret
	}

	keyring, err := credentials.CreateOCIRegistryKeyring(secrets, nil)
	if err != nil {
		return nil, err
	}

	ociClient, err := oci.NewClient(log, oci.WithResolver{Resolver: keyring})
	if err != nil {
		return nil, err
	}

	blueprintsRegistry, err := blueprintsoci.NewWithOCIClient(log, ociClient)
	if err != nil {
		return nil, errors.Wrap(err, "unable to setup blueprints registry")
	}
	componentsRegistry, err := componentsregistry.NewOCIRegistryWithOCIClient(log, ociClient)
	if err != nil {
		return nil, errors.Wrap(err, "unable to setup components registry")
	}

	return &registries{
		blueprintsRegistry: blueprintsRegistry,
		componentsRegistry: componentsRegistry,
	}, nil
}
