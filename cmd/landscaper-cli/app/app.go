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

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/gardener/landscaper/cmd/landscaper-cli/app/blueprints"
	"github.com/gardener/landscaper/cmd/landscaper-cli/app/componentdescriptor"
	"github.com/gardener/landscaper/cmd/landscaper-cli/app/config"
	"github.com/gardener/landscaper/pkg/logger"
	"github.com/gardener/landscaper/pkg/version"

	"github.com/spf13/cobra"
)

func NewLandscaperCliCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "landscaper-cli",
		Short: "landscaper cli",
		PreRun: func(cmd *cobra.Command, args []string) {
			log, err := logger.NewCliLogger()
			if err != nil {
				fmt.Println("unable to setup logger")
				fmt.Println(err.Error())
				os.Exit(1)
			}
			logger.SetLogger(log)
		},
	}

	logger.InitFlags(cmd.Flags())

	cmd.AddCommand(NewVersionCommand())
	cmd.AddCommand(config.NewConfigCommand(ctx))
	cmd.AddCommand(blueprints.NewBlueprintsCommand(ctx))
	cmd.AddCommand(componentdescriptor.NewComponentsCommand(ctx))

	return cmd
}

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "v",
		Run: func(cmd *cobra.Command, args []string) {
			v := version.Get()
			fmt.Printf("%#v", v)
		},
	}
}
