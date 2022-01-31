// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"context"
	"flag"
	"testing"

	"github.com/gardener/landscaper/test/integration/deployitems"
	"github.com/gardener/landscaper/test/integration/installations"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/pkg/utils/simplelogger"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/integration/core"
	"github.com/gardener/landscaper/test/integration/deployers"
	"github.com/gardener/landscaper/test/integration/executions"
	"github.com/gardener/landscaper/test/integration/tutorial"
	"github.com/gardener/landscaper/test/integration/webhook"
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

	logger := simplelogger.NewLogger()

	opts.RootPath = "../../"
	f, err := framework.New(logger, opts)
	if err != nil {
		t.Fatal(err)
	}
	d := framework.NewDumper(f.Log(), f.Client, f.ClientSet, f.LsNamespace)
	err = f.WaitForSystemComponents(ctx)
	if err != nil {
		if derr := d.Dump(ctx); derr != nil {
			f.Log().Logf("error during dump: %s", derr.Error())
		}
		t.Fatal(err)
	}

	tutorial.RegisterTests(f)
	webhook.RegisterTests(f)
	core.RegisterTests(f)
	deployers.RegisterTests(f)
	deployitems.RegisterTests(f)
	installations.RegisterTests(f)
	executions.RegisterTests(f)

	AfterSuite(func() {
		f.Cleanup.Run()
	})

	RunSpecs(t, "Landscaper Integration Test Suite")
}
