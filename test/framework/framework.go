// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsscheme "github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/utils/simplelogger"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

type Options struct {
	fs             *flag.FlagSet
	KubeconfigPath string
	RootPath       string
	LsNamespace    string
}

// AddFlags registers the framework related flags
func (o *Options) AddFlags(fs *flag.FlagSet) {
	if fs == nil {
		fs = flag.CommandLine
	}
	if kFlag := fs.Lookup("kubeconfig"); kFlag == nil {
		fs.String("kubeconfig", "", "Path to the kubeconfig")
	}
	fs.StringVar(&o.LsNamespace, "ls-namespace", "", "Namespace where the landscaper controller is running")
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

// Framework is the Landscaper test framework to execute tests.
// Also includes some helper functions.
type Framework struct {
	logger simplelogger.Logger
	// RootPath is the filepath to the root of the landscaper repository
	RootPath string
	// RestConfig is the kubernetes rest config for the test cluster
	RestConfig *rest.Config
	// Client is the kubernetes client to interact with the test cluster
	Client client.Client
	// ClientSet is the kubernetes clientset to interact with the test cluster.
	ClientSet kubernetes.Interface
	// Cleanups contains all cleanup handles that are executed in the after suite
	Cleanup *Cleanup
	// LsNamespace defines the namespace where the landscaper controlplane components are deployed.
	// All functionality like waiting for the components to be ready or log dump is not available
	// if left empty.
	LsNamespace string
}

func New(logger simplelogger.Logger, cfg *Options) (*Framework, error) {
	if err := cfg.Complete(); err != nil {
		return nil, err
	}
	f := &Framework{
		logger:      logger,
		RootPath:    cfg.RootPath,
		LsNamespace: cfg.LsNamespace,
		Cleanup:     &Cleanup{},
	}

	var err error
	f.RestConfig, err = clientcmd.BuildConfigFromFlags("", cfg.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to parse kubeconfig: %w", err)
	}
	f.Client, err = client.New(f.RestConfig, client.Options{
		Scheme: lsscheme.LandscaperScheme,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to build kubernetes client: %w", err)
	}
	f.ClientSet, err = kubernetes.NewForConfig(f.RestConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to build kubernetes clientset: %w", err)
	}
	return f, nil
}

// Log returns the default logger
func (f *Framework) Log() simplelogger.Logger {
	return f.logger
}

// TestLog returns a new testlogger that logs to the ginkgo managed writer
func (f *Framework) TestLog() simplelogger.Logger {
	return simplelogger.NewLoggerFromWriter(ginkgo.GinkgoWriter)
}

// WaitForSystemComponents waits for all system component of the landscaper to be ready
func (f *Framework) WaitForSystemComponents(ctx context.Context) error {
	if len(f.LsNamespace) == 0 {
		return nil
	}
	f.logger.WithTimestamp().Logf("Waiting for Landscaper components to be ready in %s", f.LsNamespace)
	// get all deployments
	deploymentList := &appsv1.DeploymentList{}

	if err := f.Client.List(ctx, deploymentList,
		client.InNamespace(f.LsNamespace),
		client.HasLabels{lsv1alpha1.LandscaperComponentLabelName}); err != nil {
		return err
	}

	for _, deployment := range deploymentList.Items {
		if err := utils.WaitForDeploymentToBeReady(ctx, f.Log(), f.Client, client.ObjectKeyFromObject(&deployment), 10*time.Minute); err != nil {
			return err
		}
	}
	return nil
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
	dumper := NewDumper(f.logger, f.Client, f.ClientSet, f.LsNamespace)
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
