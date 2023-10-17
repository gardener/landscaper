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
	"path/filepath"

	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/landscaper/apis/config"

	"github.com/gardener/landscaper/pkg/components/cache/blueprint"
	"github.com/gardener/landscaper/pkg/deployerlegacy"

	"github.com/gardener/landscaper/pkg/components/registries"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containercore "github.com/gardener/landscaper/apis/deployer/container"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/deployer/container"
	"github.com/gardener/landscaper/pkg/deployer/container/state"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/utils"
)

// Run downloads the import config, the component descriptor and the blob content
// to the paths defined by the env vars.
// It also creates all needed directories.
func Run(ctx context.Context, fs vfs.FileSystem) error {
	log, ctx := logging.FromContextOrNew(ctx, nil)
	log = log.WithName("container").WithName("init")
	opts := &options{}
	opts.Complete()
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
			Scheme: api.LandscaperScheme,
		})
		if err != nil {
			log.Error(err, "Unable to build kubernetes client")
			return false, nil
		}
		return true, nil
	}); err != nil {
		return err
	}
	return run(ctx, opts, kubeClient, fs)
}

func run(ctx context.Context, opts *options, kubeClient client.Client, fs vfs.FileSystem) error {
	log, ctx := logging.FromContextOrNew(ctx, nil)
	providerConfigBytes, err := vfs.ReadFile(fs, opts.ConfigurationFilePath)
	if err != nil {
		return fmt.Errorf("unable to read provider configuration: %w", err)
	}
	providerConfig := &containerv1alpha1.ProviderConfiguration{}
	decoder := serializer.NewCodecFactory(container.Scheme).UniversalDecoder()
	if _, _, err := decoder.Decode(providerConfigBytes, nil, providerConfig); err != nil {
		return err
	}

	// create all directories
	log.Info("Creating directories")
	if err := fs.MkdirAll(path.Dir(opts.ImportsFilePath), os.ModePerm); err != nil {
		return err
	}
	if err := fs.MkdirAll(path.Dir(opts.ExportsFilePath), os.ModePerm); err != nil {
		return err
	}
	if err := fs.MkdirAll(path.Dir(opts.ComponentDescriptorFilePath), os.ModePerm); err != nil {
		return err
	}
	if err := fs.MkdirAll(path.Dir(opts.TargetFilePath), os.ModePerm); err != nil {
		return err
	}
	if err := fs.MkdirAll(opts.ContentDirPath, os.ModePerm); err != nil {
		return err
	}
	if err := fs.MkdirAll(opts.StateDirPath, os.ModePerm); err != nil {
		return err
	}
	log.Info("All directories have been successfully created")

	var (
		cdReference    *lsv1alpha1.ComponentDescriptorReference
		registryAccess model.RegistryAccess
	)

	if providerConfig.ComponentDescriptor != nil {
		cdReference = deployerlegacy.GetReferenceFromComponentDescriptorDefinition(providerConfig.ComponentDescriptor)
		if cdReference == nil {
			return fmt.Errorf("no inline component descriptor or reference found")
		}

		var secrets []string
		err := vfs.Walk(fs, opts.RegistrySecretBasePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || info.Name() != corev1.DockerConfigJsonKey {
				return nil
			}

			secrets = append(secrets, path)

			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("unable to add local registry pull secrets: %w", err)
		}
		ociconfig := &config.OCIConfiguration{
			ConfigFiles:        secrets,
			Cache:              nil,
			AllowPlainHttp:     false,
			InsecureSkipVerify: false,
		}
		registryAccess, err = registries.GetFactory(opts.UseOCM).NewRegistryAccess(ctx, fs, nil, nil, nil, ociconfig, providerConfig.ComponentDescriptor.Inline)
		if err != nil {
			return err
		}

		if err := fetchComponentDescriptor(ctx, registryAccess, opts, fs, cdReference); err != nil {
			return fmt.Errorf("unable to fetch component descriptor: %w", err)
		}
	}

	if providerConfig.Blueprint != nil {
		log.Info("Getting blueprint content")
		// setup a temporary blueprint store
		store, err := blueprint.DefaultStore(memoryfs.New())
		if err != nil {
			return fmt.Errorf("unable to setup default blueprint store: %w", err)
		}
		blueprint.SetStore(store)
		contentFS, err := projectionfs.New(fs, opts.ContentDirPath)
		if err != nil {
			return fmt.Errorf("unable to create projection filesystem for path %s: %w", opts.ContentDirPath, err)
		}

		bp, err := blueprints.Resolve(ctx, registryAccess, cdReference, *providerConfig.Blueprint)
		if err != nil {
			return fmt.Errorf("unable to resolve blueprint and component descriptor: %w", err)
		}
		if err := utils.CopyFS(bp.Fs, contentFS, "/", "/"); err != nil {
			return fmt.Errorf("unable to copy blueprint to content dir path: %w", err)
		}
	}

	if providerConfig.ImportValues != nil {
		log.Info("Writing import values")
		if err := vfs.WriteFile(fs, opts.ImportsFilePath, providerConfig.ImportValues, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write imported values: %w", err)
		}
	}

	// copy target to shared volume
	targetSource := filepath.Join(containercore.TargetInitDir, containercore.TargetFileName)
	targetContent, err := vfs.ReadFile(fs, targetSource)
	if err != nil {
		return fmt.Errorf("error reading target content from '%s': %w", targetSource, err)
	}
	if err := vfs.WriteFile(fs, opts.TargetFilePath, targetContent, os.ModePerm); err != nil {
		return fmt.Errorf("error writing target content to '%s': %w", opts.TargetFilePath, err)
	}
	log.Info("Copied target content to shared volume.")

	log.Info("Restoring state")
	if err := state.New(kubeClient, opts.podNamespace, opts.DeployItemKey, opts.StateDirPath).WithFs(fs).Restore(ctx); err != nil {
		return err
	}
	log.Info("State has been successfully restored")

	return nil
}

func fetchComponentDescriptor(
	ctx context.Context,
	registryAccess model.RegistryAccess,
	opts *options,
	fs vfs.FileSystem,
	cdRef *lsv1alpha1.ComponentDescriptorReference) error {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	if cdRef == nil || cdRef.RepositoryContext == nil {
		return nil
	}

	log.Info("Resolving component descriptor")
	componentVersion, err := registryAccess.GetComponentVersion(ctx, cdRef)
	if err != nil {
		return fmt.Errorf("unable to resolve component descriptor for ref %v %s:%s: %w", string(cdRef.RepositoryContext.Raw), cdRef.ComponentName, cdRef.Version, err)
	}

	resolvedComponentVersions, err := model.GetTransitiveComponentReferences(ctx,
		componentVersion,
		cdRef.RepositoryContext,
		nil)
	if err != nil {
		return errors.Wrapf(err, "unable to resolve transitive component references for component version %s:%s", componentVersion.GetName(), componentVersion.GetVersion())
	}

	resolvedComponents, err := model.ConvertComponentVersionList(resolvedComponentVersions)
	if err != nil {
		return errors.Wrapf(err, "unable to convert list of component references of component version %s:%s", componentVersion.GetName(), componentVersion.GetVersion())
	}

	cdListJSONBytes, err := json.Marshal(resolvedComponents)
	if err != nil {
		return errors.Wrap(err, "unable to unmarshal mapped component descriptor")
	}
	if err := vfs.WriteFile(fs, opts.ComponentDescriptorFilePath, cdListJSONBytes, os.ModePerm); err != nil {
		return errors.Wrapf(err, "unable to write mapped component descriptor to file %s", opts.ComponentDescriptorFilePath)
	}
	return nil
}
