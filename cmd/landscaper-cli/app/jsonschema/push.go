// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package jsonschema

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/landscaper/cmd/landscaper-cli/app/constants"
	"github.com/gardener/landscaper/pkg/landscaper/jsonschema"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	"github.com/gardener/landscaper/pkg/logger"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

type pushOptions struct {
	// ref is the oci artifact uri reference to the uploaded component descriptor
	ref string
	// jsonschemaPath defines the path to the jsonschema
	jsonschemaPath string
	// cacheDir defines the oci cache directory
	cacheDir string
}

// NewPushCommand creates a new definition command to push definitions
func NewPushCommand(ctx context.Context) *cobra.Command {
	opts := &pushOptions{}
	cmd := &cobra.Command{
		Use:     "push",
		Args:    cobra.ExactArgs(4),
		Example: "landscapercli js push [refurl] [path to jsonschema]",
		Short:   "command to interact with a component descriptor stored an oci registry",
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

	data, err := ioutil.ReadFile(o.jsonschemaPath)
	if err != nil {
		return err
	}

	desc := ocispecv1.Descriptor{
		MediaType: jsonschema.JSONSchemaMediaType,
		Digest:    digest.FromBytes(data),
		Size:      int64(len(data)),
	}
	if err := cache.Add(desc, ioutil.NopCloser(bytes.NewBuffer(data))); err != nil {
		return fmt.Errorf("unable to add layer to internal cache: %w", err)
	}

	dummyDesc, err := cdutils.AddDummyDescriptor(cache)
	if err != nil {
		return fmt.Errorf("unable to add dummy descriptor: %w", err)
	}

	manifest := &ocispecv1.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Config:    dummyDesc,
		Layers: []ocispecv1.Descriptor{
			desc,
		},
	}

	ociClient, err := oci.NewClient(log, oci.WithCache{Cache: cache}, oci.WithKnownMediaType(jsonschema.JSONSchemaMediaType))
	if err != nil {
		return err
	}

	return ociClient.PushManifest(ctx, o.ref, manifest)
}

func (o *pushOptions) Complete(args []string) error {
	o.ref = args[0]
	o.jsonschemaPath = args[1]

	landscaperCliHomeDir, err := constants.LandscaperCliHomeDir()
	if err != nil {
		return err
	}
	o.cacheDir = filepath.Join(landscaperCliHomeDir, "components")
	if err := os.MkdirAll(o.cacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create cache directory %s: %w", o.cacheDir, err)
	}

	return o.Validate()
}

// Validate validates push options
func (o *pushOptions) Validate() error {
	if len(o.ref) == 0 {
		return errors.New("the reference must be defined")
	}
	if len(o.jsonschemaPath) == 0 {
		return errors.New("a path to the jsonschema must be defined")
	}
	if len(o.cacheDir) == 0 {
		return errors.New("a oci cache directory must be defined")
	}

	_, err := os.Stat(o.jsonschemaPath)
	if err != nil {
		return err
	}

	return nil
}

func (o *pushOptions) AddFlags(fs *pflag.FlagSet) {}
