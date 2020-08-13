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
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/yaml"

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
}

// NewGetDefinitionsCommand shows definitions and their configuration.
func NewGetDefinitionsCommand(ctx context.Context) *cobra.Command {
	opts := &showOptions{}
	cmd := &cobra.Command{
		Use:     "get",
		Args:    cobra.MinimumNArgs(1),
		Example: "landscapercli definitions get [ref]",
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
	cache, err := cache.NewCache(log)
	if err != nil {
		return err
	}

	ociClient, err := oci.NewClient(log, oci.WithCache{Cache: cache})
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

	memFS := afero.NewMemMapFs()
	if err := utils.ExtractTarGzip(&data, memFS, "/"); err != nil {
		return err
	}

	defData, err := afero.ReadFile(memFS, filepath.Join("/", lsv1alpha1.ComponentDefinitionPath))
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
	// todo: validate args
	o.ref = args[0]
	return nil
}

func (o *showOptions) AddFlags(fs *pflag.FlagSet) {

}
