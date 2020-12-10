// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/landscaper/cmd/landscaper-cli/app/constants"
	"github.com/gardener/landscaper/pkg/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/apis/core/validation"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints/bputils"
	"github.com/gardener/landscaper/pkg/logger"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

type pushOptions struct {
	// ref is the oci reference where the definition should eb uploaded.
	ref string
	// allowPlainHttp allows the fallback to http if the oci registry does not support https
	allowPlainHttp bool

	// blueprintPath is the path to the directory containing the definition.
	blueprintPath string
	// cacheDir defines the oci cache directory
	cacheDir string
}

// NewPushCommand creates a new blueprint command to push blueprints
func NewPushCommand(ctx context.Context) *cobra.Command {
	opts := &pushOptions{}
	cmd := &cobra.Command{
		Use:     "push",
		Args:    cobra.ExactArgs(2),
		Example: "landscapercli blueprints push [ref] [path to Blueprint directory]",
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

			fmt.Printf("Successfully uploaded %s\n", opts.ref)
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func (o *pushOptions) run(ctx context.Context, log logr.Logger) error {
	cache, err := cache.NewCache(log, cache.WithBasePath(o.cacheDir))
	if err != nil {
		return err
	}

	defManifest, err := bputils.BuildNewDefinition(cache, osfs.New(), o.blueprintPath)
	if err != nil {
		return err
	}

	ociClient, err := oci.NewClient(log,
		oci.WithCache{Cache: cache},
		oci.WithKnownMediaType(lsv1alpha1.BlueprintArtifactsMediaType),
		oci.AllowPlainHttp(o.allowPlainHttp))
	if err != nil {
		return err
	}

	return ociClient.PushManifest(ctx, o.ref, defManifest)
}

func (o *pushOptions) Complete(args []string) error {
	o.ref = args[0]
	o.blueprintPath = args[1]

	landscaperCliHomeDir, err := constants.LandscaperCliHomeDir()
	if err != nil {
		return err
	}
	o.cacheDir = filepath.Join(landscaperCliHomeDir, "components")
	if err := os.MkdirAll(o.cacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create cache directory %s: %w", o.cacheDir, err)
	}

	if len(o.cacheDir) == 0 {
		return errors.New("a oci cache directory must be defined")
	}
	return o.Validate()
}

// Validate validates push options
func (o *pushOptions) Validate() error {
	data, err := ioutil.ReadFile(filepath.Join(o.blueprintPath, lsv1alpha1.BlueprintFileName))
	if err != nil {
		return err
	}
	blueprint := &core.Blueprint{}
	if _, _, err := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder().Decode(data, nil, blueprint); err != nil {
		return err
	}

	blueprintFs, err := projectionfs.New(osfs.New(), o.blueprintPath)
	if err != nil {
		return fmt.Errorf("unable to construct blueprint filesystem: %w", err)
	}
	if errList := validation.ValidateBlueprint(blueprintFs, blueprint); len(errList) != 0 {
		return errList.ToAggregate()
	}

	return nil
}

func (o *pushOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.allowPlainHttp, "allow-plain-http", false, "allows the fallback to http if the oci registry does not support https")
}
