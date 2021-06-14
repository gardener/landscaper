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

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/landscaper/installations"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"

	"github.com/gardener/component-cli/ociclient/credentials"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/deployer/container"
	"github.com/gardener/landscaper/pkg/deployer/container/state"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
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
			Scheme: api.LandscaperScheme,
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

	var (
		cdReference *lsv1alpha1.ComponentDescriptorReference
		cdResolver  ctf.ComponentResolver
	)

	if providerConfig.ComponentDescriptor != nil {
		cdReference = installations.GetReferenceFromComponentDescriptorDefinition(providerConfig.ComponentDescriptor)
		if cdReference == nil {
			return fmt.Errorf("no inline component descriptor or reference found")
		}

		ociClient, err := createOciClientFromDockerAuthConfig(ctx, log, fs, opts.RegistrySecretBasePath)
		if err != nil {
			return err
		}

		cdResolver, err = componentsregistry.NewOCIRegistryWithOCIClient(log, ociClient, providerConfig.ComponentDescriptor.Inline)
		if err != nil {
			return errors.Wrap(err, "unable to setup components registry")
		}

		if err := fetchComponentDescriptor(ctx, log, cdResolver, opts, fs, providerConfig); err != nil {
			return fmt.Errorf("unable to fetch component descriptor: %w", err)
		}
	}

	if providerConfig.Blueprint != nil {
		log.Info("get blueprint content")
		contentFS, err := projectionfs.New(fs, opts.ContentDirPath)
		if err != nil {
			return errors.Wrapf(err, "unable to create projection filesystem for path %s", opts.ContentDirPath)
		}

		if _, err := blueprints.Resolve(ctx, cdResolver, cdReference, *providerConfig.Blueprint, contentFS); err != nil {
			return fmt.Errorf("unable to resolve blueprint and component descriptor")
		}
	}

	if providerConfig.ImportValues != nil {
		log.Info("write import values")
		if err := vfs.WriteFile(fs, opts.ImportsFilePath, providerConfig.ImportValues, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write imported values: %w", err)
		}
	}

	log.Info("restore state")
	if err := state.New(log, kubeClient, opts.podNamespace, opts.DeployItemKey, opts.StateDirPath).WithFs(fs).Restore(ctx); err != nil {
		return err
	}
	log.Info("state has been successfully restored")

	return nil
}

func fetchComponentDescriptor(
	ctx context.Context,
	log logr.Logger,
	resolver ctf.ComponentResolver,
	opts *options,
	fs vfs.FileSystem,
	providerConfig *containerv1alpha1.ProviderConfiguration) error {

	cdRef := installations.GetReferenceFromComponentDescriptorDefinition(providerConfig.ComponentDescriptor)
	if cdRef == nil || cdRef.RepositoryContext == nil {
		return nil
	}

	log.Info("get component descriptor")
	cd, err := resolver.Resolve(ctx, cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
	if err != nil {
		return fmt.Errorf("unable to resolve component descriptor for ref %v %s:%s: %w", string(cdRef.RepositoryContext.Raw), cdRef.ComponentName, cdRef.Version, err)
	}

	resolvedComponents, err := cdutils.ResolveToComponentDescriptorList(ctx, resolver, *cd)
	if err != nil {
		return errors.Wrapf(err, "unable to resolve component descriptor references for ref %#v", providerConfig.Blueprint)
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

// todo: add retries
func createOciClientFromDockerAuthConfig(_ context.Context, log logr.Logger, fs vfs.FileSystem, registryPullSecretsDir string) (ociclient.Client, error) {
	var secrets []string
	err := vfs.Walk(fs, registryPullSecretsDir, func(path string, info os.FileInfo, err error) error {
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
		return nil, fmt.Errorf("unable to add local registry pull secrets: %w", err)
	}

	keyring, err := credentials.CreateOCIRegistryKeyringFromFilesystem(nil, secrets, fs)
	if err != nil {
		return nil, err
	}

	ociClient, err := ociclient.NewClient(log, ociclient.WithKeyring(keyring))
	if err != nil {
		return nil, err
	}

	return ociClient, err
}
