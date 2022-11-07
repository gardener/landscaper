// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gardener/landscaper/cmd/target-sync-controller/app"
)

func main() {
	ctx := context.Background()
	defer ctx.Done()
	cmd := app.NewTargetSyncControllerCommand(ctx)

	if err := cmd.Execute(); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}
