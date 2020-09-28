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
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/landscaper/cmd/landscaper-cli/app/constants"
	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	blueprintsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/registry/blueprints/bputils"
	"github.com/gardener/landscaper/pkg/logger"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

type pushOptions struct {
	// ref is the oci reference where the definition should eb uploaded.
	ref string

	// blueprintPath is the path to the directory containing the definition.
	blueprintPath string
	// cacheDir defines the oci cache directory
	cacheDir string
}

// NewPushCommand creates a new definition command to push definitions
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

	ociClient, err := oci.NewClient(log, oci.WithCache{Cache: cache}, oci.WithKnownMediaType(blueprintsregistry.ComponentDefinitionConfigMediaType))
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
	data, err := ioutil.ReadFile(filepath.Join(o.blueprintPath, lsv1alpha1.BlueprintFilePath))
	if err != nil {
		return err
	}
	blueprint := &lsv1alpha1.Blueprint{}
	if _, _, err := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder().Decode(data, nil, blueprint); err != nil {
		return err
	}

	// todo: validate references exist
	return nil
}

func (o *pushOptions) AddFlags(fs *pflag.FlagSet) {}
