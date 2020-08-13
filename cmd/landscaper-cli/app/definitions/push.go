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

package definitions

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/json"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/registry"
	lsoci "github.com/gardener/landscaper/pkg/landscaper/registry/oci"
	"github.com/gardener/landscaper/pkg/logger"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

type pushOptions struct {
	// ref is the oci reference where the definition should eb uploaded.
	ref string

	// definitionPath is the path to the directory containing the definition.
	definitionPath string

	// definitionPath is the path to the string
	definition *lsv1alpha1.Blueprint
}

// NewPushDefinitionsCommand creates a new definition command to push definitions
func NewPushDefinitionsCommand(ctx context.Context) *cobra.Command {
	opts := &pushOptions{}
	cmd := &cobra.Command{
		Use:     "push",
		Args:    cobra.ExactArgs(2),
		Example: "landscapercli definitions push [ref] [path to Blueprint directory]",
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
	cache, err := cache.NewCache(log)
	if err != nil {
		return err
	}

	defManifest, err := lsoci.BuildNewDefinition(cache, afero.NewOsFs(), o.definitionPath)
	if err != nil {
		return err
	}

	ociClient, err := oci.NewClient(log, oci.WithCache{Cache: cache}, oci.WithKnownMediaType(lsoci.ComponentDefinitionConfigMediaType))
	if err != nil {
		return err
	}

	return ociClient.PushManifest(ctx, o.ref, defManifest)
}

func (o *pushOptions) Complete(args []string) error {
	// todo: validate args
	o.ref = args[0]
	o.definitionPath = args[1]

	data, err := ioutil.ReadFile(filepath.Join(o.definitionPath, lsv1alpha1.ComponentDefinitionPath))
	if err != nil {
		return err
	}
	o.definition = &lsv1alpha1.Blueprint{}
	if _, _, err := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder().Decode(data, nil, o.definition); err != nil {
		return err
	}

	// automatically add default component descriptor is none is defined
	if _, err := os.Stat(filepath.Join(o.definitionPath, lsv1alpha1.ComponentDefinitionComponentDescriptorPath)); err != nil {
		vName, err := registry.ParseDefinitionRef(o.ref)
		if err != nil {
			return err
		}
		ociComponent := &cdv2.OCIComponent{
			ComponentMetadata: cdv2.ComponentMetadata{
				Type:    cdv2.OCIComponentType,
				Name:    o.definition.Name,
				Version: o.definition.Version,
			},
			Repository: vName.Name,
		}

		cd := cdv2.ComponentDescriptor{
			Metadata:   cdv2.Metadata{Version: cdv2.SchemaVersion},
			Components: cdv2.ResolvableComponentList{ociComponent},
		}

		data, err := json.Marshal(cd)
		if err != nil {
			return fmt.Errorf("unable to parse automatically constructed component descriptor: %w", err)
		}

		if err := ioutil.WriteFile(filepath.Join(o.definitionPath, lsv1alpha1.ComponentDefinitionComponentDescriptorPath), data, os.ModePerm); err != nil {
			return err
		}
	}

	return o.Validate()
}

// Validate validates push options
func (o *pushOptions) Validate() error {
	// require a component descriptor
	if _, err := os.Stat(filepath.Join(o.definitionPath, lsv1alpha1.ComponentDefinitionComponentDescriptorPath)); err != nil {
		return fmt.Errorf("ComponentDescriptor is required at %s", filepath.Join(o.definitionPath, lsv1alpha1.ComponentDefinitionComponentDescriptorPath))
	}

	return nil
}

func (o *pushOptions) AddFlags(fs *pflag.FlagSet) {}
