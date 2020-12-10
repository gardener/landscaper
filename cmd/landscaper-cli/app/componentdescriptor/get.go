// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentdescriptor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/cmd/landscaper-cli/app/constants"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/logger"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

type showOptions struct {
	// baseUrl is the oci registry where the component is stored.
	baseUrl string
	// componentName is the unique name of the component in the registry.
	componentName string
	// version is the component version in the oci registry.
	version string
	// allowPlainHttp allows the fallback to http if the oci registry does not support https
	allowPlainHttp bool

	// cacheDir defines the oci cache directory
	cacheDir string
}

// NewGetCommand shows definitions and their configuration.
func NewGetCommand(ctx context.Context) *cobra.Command {
	opts := &showOptions{}
	cmd := &cobra.Command{
		Use:     "get",
		Args:    cobra.ExactArgs(3),
		Example: "landscapercli cd get [baseurl] [componentname] [version]",
		Short:   "fetch the component descriptor from a oci registry",
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			if err := opts.run(ctx, logger.Log); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func (o *showOptions) run(ctx context.Context, log logr.Logger) error {
	cache, err := cache.NewCache(log, cache.WithBasePath(o.cacheDir))
	if err != nil {
		return err
	}

	ociClient, err := oci.NewClient(log, oci.WithCache{Cache: cache}, oci.AllowPlainHttp(o.allowPlainHttp))
	if err != nil {
		return err
	}

	componentRegistry, err := componentsregistry.NewOCIRegistryWithOCIClient(log, ociClient)
	if err != nil {
		return err
	}

	repoCtx := cdv2.RepositoryContext{
		Type:    cdv2.OCIRegistryType,
		BaseURL: o.baseUrl,
	}
	ociRef, err := cdoci.OCIRef(repoCtx, o.componentName, o.version)
	if err != nil {
		return fmt.Errorf("invalid component reference: %w", err)
	}
	cd, _, err := componentRegistry.Resolve(ctx, repoCtx, o.componentName, o.version)
	if err != nil {
		return fmt.Errorf("unable to to fetch component descriptor %s: %w", ociRef, err)
	}

	out, err := yaml.Marshal(cd)
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

func (o *showOptions) Complete(args []string) error {
	// todo: validate args
	o.baseUrl = args[0]
	o.componentName = args[1]
	o.version = args[2]

	landscaperCliHomeDir, err := constants.LandscaperCliHomeDir()
	if err != nil {
		return err
	}
	o.cacheDir = filepath.Join(landscaperCliHomeDir, "components")
	if err := os.MkdirAll(o.cacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create cache directory %s: %w", o.cacheDir, err)
	}

	if len(o.baseUrl) == 0 {
		return errors.New("the base url must be defined")
	}
	if len(o.componentName) == 0 {
		return errors.New("a component name must be defined")
	}
	if len(o.version) == 0 {
		return errors.New("a component's version must be defined")
	}
	if len(o.cacheDir) == 0 {
		return errors.New("a cache directory must be defined")
	}
	return nil
}

func (o *showOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.allowPlainHttp, "allow-plain-http", false, "allows the fallback to http if the oci registry does not support https")
}
