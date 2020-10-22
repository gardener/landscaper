// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
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
	artifactsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/artifacts"
	blueprintsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/credentials"
)

// Run downloads the import config, the component descriptor and the blob content
// to the paths defined by the env vars.
// It also creates all needed directories.
func Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
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
	return run(ctx, log, opts, kubeClient, fs)
}

func run(ctx context.Context, log logr.Logger, opts *options, kubeClient client.Client, fs vfs.FileSystem) error {
	providerConfigBytes, err := vfs.ReadFile(fs, opts.ConfigurationFilePath)
	if err != nil {
		return fmt.Errorf("unable to read provider configuration: %w", err)
	}
	providerConfig := &containerv1alpha1.ProviderConfiguration{}
	decoder := serializer.NewCodecFactory(container.Scheme).UniversalDecoder()
	if _, _, err := decoder.Decode(providerConfigBytes, nil, providerConfig); err != nil {
		return err
	}

	regAcc, err := createRegistryFromDockerAuthConfig(ctx, log, kubeClient, providerConfig.RegistryPullSecrets)
	if err != nil {
		return err
	}

	// create all directories
	log.Info("create directories")
	if err := fs.MkdirAll(path.Dir(opts.ImportsFilePath), os.ModePerm); err != nil {
		return err
	}
	if err := fs.MkdirAll(path.Dir(opts.ExportsFilePath), os.ModePerm); err != nil {
		return err
	}
	if err := fs.MkdirAll(path.Dir(opts.ComponentDescriptorFilePath), os.ModePerm); err != nil {
		return err
	}
	if err := fs.MkdirAll(opts.ContentDirPath, os.ModePerm); err != nil {
		return err
	}
	if err := fs.MkdirAll(opts.StateDirPath, os.ModePerm); err != nil {
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
			if err := vfs.WriteFile(fs, opts.ComponentDescriptorFilePath, cdListJSONBytes, os.ModePerm); err != nil {
				return errors.Wrapf(err, "unable to write mapped component descriptor to file %s", opts.ComponentDescriptorFilePath)
			}
		}

		log.Info("get blueprint content")
		contentFS, err := projectionfs.New(fs, opts.ContentDirPath)
		if err != nil {
			return errors.Wrapf(err, "unable to create projection filesystem for path %s", opts.ContentDirPath)
		}
		if providerConfig.Blueprint.Reference != nil {
			log.Info(fmt.Sprintf("fetching blueprint for %#v", providerConfig.Blueprint.Reference))
			// resolve is only used to download the blueprint's content to the filesystem
			_, err = blueprints.Resolve(ctx, regAcc, *providerConfig.Blueprint, contentFS)
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
			if err := utils.CopyFS(blueprintFs, contentFS, "/", "/"); err != nil {
				return fmt.Errorf("unabel to copy inline blueprint filesystem to container filesystem: %w", err)
			}
		}
	}

	if providerConfig.ImportValues != nil {
		log.Info("write import values")
		if err := vfs.WriteFile(fs, opts.ImportsFilePath, providerConfig.ImportValues, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write imported values: %w", err)
		}
	}

	log.Info("restore state")
	if err := state.New(log, kubeClient, opts.DeployItemKey, opts.StateDirPath).Restore(ctx, fs); err != nil {
		return err
	}
	log.Info("state has been successfully restored")

	return nil
}

// registries is a internal struct that implements the registry accessors interface
type registries struct {
	blueprintsRegistry blueprintsregistry.Registry
	componentsRegistry componentsregistry.Registry
	artifactsRegistry  artifactsregistry.Registry
}

var _ lsoperation.RegistriesAccessor = &registries{}

func (r registries) BlueprintsRegistry() blueprintsregistry.Registry {
	return r.blueprintsRegistry
}

func (r registries) ComponentsRegistry() componentsregistry.Registry {
	return r.componentsRegistry
}

func (r registries) ArtifactsRegistry() artifactsregistry.Registry {
	return r.artifactsRegistry
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
	artifactsRegistry, err := artifactsregistry.NewOCIRegistryWithOCIClient(log, ociClient)
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
		artifactsRegistry:  artifactsRegistry,
	}, nil
}
