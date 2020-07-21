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

	"github.com/spf13/cobra"
)

// NewDefinitionsCommand creates a new definitions command.
func NewDefinitionsCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "definitions",
		Aliases: []string{"defs", "definition", "def"},
		Short:   "command to interact with definitions of an oci registry",
	}

	cmd.AddCommand(NewPushDefinitionsCommand(ctx))
	cmd.AddCommand(NewGetDefinitionsCommand(ctx))

	return cmd
}
