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

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/yamlfs"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/container"
	"github.com/gardener/landscaper/pkg/deployer/container/state"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	"github.com/gardener/landscaper/pkg/utils"
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
	if err := kubeClient.Get(ctx, opts.DeployItemKey.NamespacedName(), deployItem); err != nil {
		return err
	}
	providerConfig := &containerv1alpha1.ProviderConfiguration{}
	decoder := serializer.NewCodecFactory(container.Scheme).UniversalDecoder()
	if _, _, err := decoder.Decode(deployItem.Spec.Configuration.Raw, nil, providerConfig); err != nil {
		return err
	}

	regAcc, err := createRegistryFromDockerAuthConfig(ctx, log, kubeClient, providerConfig.RegistryPullSecrets)
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

	if providerConfig.Blueprint != nil && providerConfig.Blueprint.Reference != nil {
		var (
			cd  *cdv2.ComponentDescriptor
			err error
		)

		if providerConfig.Blueprint.Reference != nil {
			log.Info("get component descriptor")
			cd, err = regAcc.ComponentsRegistry().Resolve(ctx, *providerConfig.Blueprint.Reference.RepositoryContext, providerConfig.Blueprint.Reference.ObjectMeta())
			if err != nil {
				return errors.Wrapf(err, "unable to resolve component descriptor for ref %#v", providerConfig.Blueprint)
			}
		}
		if providerConfig.Blueprint.Inline.ComponentDescriptorReference != nil {
			log.Info("get component descriptor")
			cd, err = regAcc.ComponentsRegistry().Resolve(ctx, *providerConfig.Blueprint.Inline.ComponentDescriptorReference.RepositoryContext, providerConfig.Blueprint.Inline.ComponentDescriptorReference.ObjectMeta())
			if err != nil {
				return errors.Wrapf(err, "unable to resolve component descriptor for ref %#v", providerConfig.Blueprint)
			}
		}

		if cd != nil {
			resolvedComponent, err := cdutils.ResolveEffectiveComponentDescriptor(ctx, regAcc.ComponentsRegistry(), *cd)
			if err != nil {
				return errors.Wrapf(err, "unable to resolve component descriptor references for ref %#v", providerConfig.Blueprint)
			}

			cdListJSONBytes, err := json.Marshal(resolvedComponent)
			if err != nil {
				return errors.Wrap(err, "unable to unmarshal mapped component descriptor")
			}
			if err := ioutil.WriteFile(opts.ComponentDescriptorFilePath, cdListJSONBytes, os.ModePerm); err != nil {
				return errors.Wrapf(err, "unable to write mapped component descriptor to file %s", opts.ComponentDescriptorFilePath)
			}
		}

		log.Info("get blueprint content")
		fs, err := projectionfs.New(osfs.New(), opts.ContentDirPath)
		if err != nil {
			return errors.Wrapf(err, "unable to create projection filesystem for path %s", opts.ContentDirPath)
		}
		if providerConfig.Blueprint.Reference != nil {
			log.Info(fmt.Sprintf("fetching blueprint for %#v", providerConfig.Blueprint.Reference))
			// resolve is only used to download the blueprint's content to the filesystem
			_, err = blueprints.Resolve(ctx, regAcc, *providerConfig.Blueprint, fs)
			if err != nil {
				return fmt.Errorf("unable to fetch blueprint from registry: %w", err)
			}
			log.Info(fmt.Sprintf("blueprint content successfully downloaded to %s", opts.ContentDirPath))
		}
		if providerConfig.Blueprint.Inline != nil {
			log.Info("using inline blueprint definition")
			blueprintFs, err := yamlfs.New(providerConfig.Blueprint.Inline.Filesystem)
			if err != nil {
				return fmt.Errorf("unable to create yaml filesystem from internal config: %w", err)
			}
			// copy yaml filesystem to conatiner filesystem
			if err := utils.CopyFS(blueprintFs, fs, "/", "/"); err != nil {
				return fmt.Errorf("unabel to copy inline blueprint filesystem to container filesystem: %w", err)
			}
		}
	}

	if providerConfig.ImportValues != nil {
		log.Info("write import values")
		if err := ioutil.WriteFile(opts.ImportsFilePath, providerConfig.ImportValues, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write imported values: %w", err)
		}
	}

	log.Info("restore state")
	if err := state.New(log, kubeClient, opts.DeployItemKey, opts.StateDirPath).Restore(ctx); err != nil {
		return err
	}
	log.Info("state has been successfully restored")

	return nil
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
func createRegistryFromDockerAuthConfig(ctx context.Context, log logr.Logger, kubeClient client.Client, registryPullSecrets []lsv1alpha1.ObjectReference) (lsoperation.RegistriesAccessor, error) {
	secrets := make([]corev1.Secret, len(registryPullSecrets))
	for i, secretRef := range registryPullSecrets {
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

	blueprintsRegistry, err := blueprintsregistry.NewOCIRegistryWithOCIClient(log, ociClient)
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
