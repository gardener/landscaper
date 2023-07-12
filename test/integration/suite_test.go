// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"context"
	"flag"
	"testing"

	"github.com/gardener/landscaper/test/integration/core"
	"github.com/gardener/landscaper/test/integration/dependencies"
	"github.com/gardener/landscaper/test/integration/deployers"
	"github.com/gardener/landscaper/test/integration/deployitems"
	"github.com/gardener/landscaper/test/integration/executions"
	"github.com/gardener/landscaper/test/integration/importexport"
	"github.com/gardener/landscaper/test/integration/inline"
	"github.com/gardener/landscaper/test/integration/installations"
	"github.com/gardener/landscaper/test/integration/rootinstallations"
	"github.com/gardener/landscaper/test/integration/subinstallations"
	"github.com/gardener/landscaper/test/integration/targets"
	"github.com/gardener/landscaper/test/integration/tutorial"
	"github.com/gardener/landscaper/test/integration/webhook"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/hack/testcluster/pkg/utils"
	"github.com/gardener/landscaper/test/framework"
)

var opts *framework.Options

func init() {
	opts = &framework.Options{}
	opts.AddFlags(flag.CommandLine)
}

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)

	ctx := context.Background()
	defer ctx.Done()

	logger := utils.NewLogger()

	logger.Logln("Create framework")
	opts.RootPath = "../../"
	f, err := framework.New(logger, opts)
	if err != nil {
		logger.Logln("Creating framework failed")
		t.Fatal(err)
	}
	d := framework.NewDumper(f.Log(), f.Client, f.ClientSet, f.LsNamespace)
	if opts.SkipWaitingForSystemComponents {
		f.Log().Logfln("Skipped waiting for system components")
	} else {
		f.Log().Logfln("Waiting for system components")
		err = f.WaitForSystemComponents(ctx)
		if err != nil {
			f.Log().Logfln("Waiting for system components failed: %s", err.Error())
			if derr := d.Dump(ctx); derr != nil {
				f.Log().Logf("error during dump: %s", derr.Error())
			}
			t.Fatal(err)
		}
	}

	if !opts.DisableCleanupBefore {
		if err := f.CleanupBeforeTestNamespaces(ctx); err != nil {
			t.Fatal(err)
		}
	}

	importexport.RegisterTests(f)
	rootinstallations.RegisterTests(f)
	subinstallations.RegisterTests(f)
	dependencies.RegisterTests(f)
	targets.RegisterTests(f)
	inline.RegisterTests(f)
	tutorial.RegisterTests(f)
	webhook.RegisterTests(f)
	core.RegisterTests(f)
	deployers.RegisterTests(f)
	deployitems.RegisterTests(f)
	installations.RegisterTests(f)
	executions.RegisterTests(f)

	AfterSuite(func() {
		f.Log().Logfln("\nStart after suite cleanup")
		f.Cleanup.Run(f.Log(), f.TestsFailed)
	})

	RunSpecs(t, "Landscaper Integration Test Suite")
}
