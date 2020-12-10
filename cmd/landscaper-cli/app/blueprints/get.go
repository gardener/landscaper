// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/cmd/landscaper-cli/app/constants"
	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/logger"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

type showOptions struct {
	// ref is the oci reference where the definition should eb uploaded.
	ref string
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
		Args:    cobra.MinimumNArgs(1),
		Example: "landscapercli blueprints get [ref]",
		Short:   "command to interact with definitions of an oci registry",
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

	manifest, err := ociClient.GetManifest(ctx, o.ref)
	if err != nil {
		return err
	}

	var data bytes.Buffer
	if err := ociClient.Fetch(ctx, o.ref, manifest.Layers[0], &data); err != nil {
		return err
	}

	memFS := memoryfs.New()
	if err := utils.ExtractTarGzip(&data, memFS, "/"); err != nil {
		return err
	}

	defData, err := vfs.ReadFile(memFS, lsv1alpha1.BlueprintFileName)
	if err != nil {
		return err
	}

	def := &lsv1alpha1.Blueprint{}
	if _, _, err := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder().Decode(defData, nil, def); err != nil {
		return err
	}

	out, err := yaml.Marshal(def)
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

func (o *showOptions) Complete(args []string) error {
	o.ref = args[0]

	landscaperCliHomeDir, err := constants.LandscaperCliHomeDir()
	if err != nil {
		return err
	}
	o.cacheDir = filepath.Join(landscaperCliHomeDir, "blueprints")
	if err := os.MkdirAll(o.cacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create cache directory %s: %w", o.cacheDir, err)
	}

	if len(o.cacheDir) == 0 {
		return errors.New("a oci cache directory must be defined")
	}
	return nil
}

func (o *showOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.allowPlainHttp, "allow-plain-http", false, "allows the fallback to http if the oci registry does not support https")
}
