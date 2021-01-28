// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"context"
	"flag"
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

type Options struct {
	fs             *flag.FlagSet
	KubeconfigPath string
	RootPath       string
}

// AddFlags registers the framework related flags
func (o *Options) AddFlags(fs *flag.FlagSet) {
	if fs == nil {
		fs = flag.CommandLine
	}
	if kFlag := fs.Lookup("kubeconfig"); kFlag == nil {
		fs.String("kubeconfig", "", "Path to the kubeconfig")
	}
	o.fs = fs
}

func (o *Options) Complete() error {
	var err error
	kFlag := o.fs.Lookup("kubeconfig")
	if kFlag == nil {
		return fmt.Errorf("kubeconfig flag not found: %w", err)
	}
	o.KubeconfigPath = kFlag.Value.String()
	return nil
}

type Framework struct {
	// RootPath is the filepath to the root of the landscaper repository
	RootPath string
	// RestConfig is the kubernetes rest config for the test cluster
	RestConfig *rest.Config
	// Client is the kubernetes client to interact with the test cluster
	Client client.Client
	// Cleanups contains all cleanup handles that are executed in the after suite
	Cleanup *Cleanup
}

func New(cfg *Options) (*Framework, error) {
	if err := cfg.Complete(); err != nil {
		return nil, err
	}
	f := &Framework{
		RootPath: cfg.RootPath,
		Cleanup:  &Cleanup{},
	}

	var err error
	f.RestConfig, err = clientcmd.BuildConfigFromFlags("", cfg.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to parse kubeconfig: %w", err)
	}
	f.Client, err = client.New(f.RestConfig, client.Options{
		Scheme: kubernetes.LandscaperScheme,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to build kubernetes client: %w", err)
	}
	return f, nil
}

type CleanupFunc func(ctx context.Context) error

// NewState creates a new state with a test namespace.
// It also returns a cleanup function that should be called when the test has finished.
func (f *Framework) NewState(ctx context.Context) (*envtest.State, CleanupFunc, error) {
	state, err := envtest.InitStateWithNamespace(ctx, f.Client)
	if err != nil {
		return nil, nil, err
	}

	// register Cleanup handle
	var handle CleanupActionHandle
	cleanupFunc := func(ctx context.Context) error {
		f.Cleanup.Remove(handle)
		return state.CleanupState(ctx, f.Client)
	}
	handle = f.Cleanup.Add(func() {
		ctx := context.Background()
		defer ctx.Done()
		gomega.Expect(cleanupFunc(ctx)).To(gomega.Succeed())
	})
	return state, cleanupFunc, err
}

// Register registers the frameworks function
// that is called by ginkgo before and after each test
func (f *Framework) Register() *Dumper {
	dumper := NewDumper(ginkgo.GinkgoWriter, f.Client)
	ginkgo.AfterEach(func() {
		if !ginkgo.CurrentGinkgoTestDescription().Failed {
			return
		}
		ctx := context.Background()
		defer ctx.Done()
		utils.ExpectNoError(dumper.Dump(ctx))
	})
	ginkgo.BeforeEach(func() {
		dumper.ClearNamespaces()
	})
	return dumper
}
