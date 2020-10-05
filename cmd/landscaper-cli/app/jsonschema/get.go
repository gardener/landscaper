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

package jsonschema

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/cmd/landscaper-cli/app/constants"
	"github.com/gardener/landscaper/pkg/landscaper/jsonschema"
	"github.com/gardener/landscaper/pkg/logger"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

type showOptions struct {
	// ref is the reference to the jsonschema in a oci registry.
	ref string
	// cacheDir defines the oci cache directory
	cacheDir string
}

// NewGetCommand shows definitions and their configuration.
func NewGetCommand(ctx context.Context) *cobra.Command {
	opts := &showOptions{}
	cmd := &cobra.Command{
		Use:     "get",
		Args:    cobra.ExactArgs(3),
		Example: "landscapercli cd get [ref]",
		Short:   "fetch the jsonschema from a oci registry",
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

	ociClient, err := oci.NewClient(log, oci.WithCache{Cache: cache})
	if err != nil {
		return err
	}

	jsonSchemaBytes, err := jsonschema.FetchFromOCIRegistry(ctx, ociClient, o.ref)
	if err != nil {
		return err
	}

	var jsonSchema interface{}
	if err := yaml.Unmarshal(jsonSchemaBytes, &jsonSchema); err != nil {
		return err
	}

	out, err := yaml.Marshal(jsonSchema)
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
	o.cacheDir = filepath.Join(landscaperCliHomeDir, "components")
	if err := os.MkdirAll(o.cacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create cache directory %s: %w", o.cacheDir, err)
	}

	if len(o.ref) == 0 {
		return errors.New("the reference must be defined")
	}
	if len(o.cacheDir) == 0 {
		return errors.New("a cache directory must be defined")
	}
	return nil
}

func (o *showOptions) AddFlags(fs *pflag.FlagSet) {}
