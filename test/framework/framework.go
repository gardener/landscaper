// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/go-logr/logr/testing"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsscheme "github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/utils/simplelogger"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

// OpenSourceRepositoryContext is the base url of the repository context for the gardener open source components.
// There all landscaper blueprints/components are available.
const OpenSourceRepositoryContext = "eu.gcr.io/gardener-project/development"

type Options struct {
	fs               *flag.FlagSet
	KubeconfigPath   string
	RootPath         string
	LsNamespace      string
	LsVersion        string
	DockerConfigPath string
	DisableCleanup   bool
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
	fs.StringVar(&o.LsVersion, "ls-version", "", "the version to use in integration tests")
	fs.StringVar(&o.DockerConfigPath, "registry-config", "", "path to the docker config file")
	fs.BoolVar(&o.DisableCleanup, "disable-cleanup", false, "skips the cleanup of resources.")
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
	// LsVersion defines the version of landscaper components to be used for the integration test
	// Will use the latest version (see VERSION) if left empty
	LsVersion string
	// DisableCleanup skips the state cleanup step
	DisableCleanup bool

	// RegistryConfig defines the oci registry config file.
	// It is expected that the configfile contains exactly one server.
	RegistryConfig *configfile.ConfigFile
	// RegistryBasePath defines the base path for the configured registry.
	// The base path is used to construct references for artifacts.
	RegistryBasePath string
	// OCIClient is a oci client that can up and download artifacts from the configured registry
	OCIClient ociclient.Client
	// OCICache is the oci store of the local oci client
	OCICache cache.Cache
}

func New(logger simplelogger.Logger, cfg *Options) (*Framework, error) {
	if err := cfg.Complete(); err != nil {
		return nil, err
	}
	f := &Framework{
		logger:         logger,
		RootPath:       cfg.RootPath,
		LsNamespace:    cfg.LsNamespace,
		LsVersion:      cfg.LsVersion,
		Cleanup:        &Cleanup{},
		DisableCleanup: cfg.DisableCleanup,
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

	if len(cfg.DockerConfigPath) != 0 {
		data, err := ioutil.ReadFile(cfg.DockerConfigPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read docker config file: %w", err)
		}
		dockerConfig := &configfile.ConfigFile{}
		if err := json.Unmarshal(data, dockerConfig); err != nil {
			return nil, fmt.Errorf("unable to decode docker config: %w", err)
		}
		if len(dockerConfig.AuthConfigs) == 0 {
			return nil, errors.New("the configured docker config must contain at least one auth config")
		}
		f.RegistryConfig = dockerConfig
		for address := range dockerConfig.AuthConfigs {
			f.RegistryBasePath = address
			break
		}

		ociKeyring, err := credentials.NewBuilder(testing.NullLogger{}).FromConfigFiles(cfg.DockerConfigPath).Build()
		if err != nil {
			return nil, fmt.Errorf("unable to build oci keyring: %w", err)
		}
		f.OCICache, err = cache.NewCache(testing.NullLogger{})
		if err != nil {
			return nil, err
		}

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		httpClient := http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}
		f.OCIClient, err = ociclient.NewClient(testing.NullLogger{},
			ociclient.WithKeyring(ociKeyring),
			ociclient.WithCache{Cache: f.OCICache},
			ociclient.WithHTTPClient(httpClient))
		if err != nil {
			return nil, fmt.Errorf("unable to build oci client: %w", err)
		}
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
		if f.DisableCleanup {
			f.Log().Logln("Skipping cleanup...")
			return nil
		}
		f.Log().Logln("Start state cleanup...")
		f.Cleanup.Remove(handle)
		t := time.Minute
		return state.CleanupState(ctx, f.Client, &t)
	}
	if !f.DisableCleanup {
		handle = f.Cleanup.Add(func() {
			ctx := context.Background()
			defer ctx.Done()
			gomega.Expect(cleanupFunc(ctx)).To(gomega.Succeed())
		})
	}
	return state, cleanupFunc, err
}

// Register registers the frameworks function
// that is called by ginkgo before and after each test
func (f *Framework) Register() *Dumper {
	dumper := NewDumper(f.logger, f.Client, f.ClientSet, f.LsNamespace)
	ginkgo.BeforeEach(func() {
		dumper.startTime = time.Now()
	})
	ginkgo.AfterEach(func() {
		dumper.endTime = time.Now()
		if !ginkgo.CurrentGinkgoTestDescription().Failed {
			return
		}
		ctx := context.Background()
		defer ctx.Done()
		utils.ExpectNoError(dumper.Dump(ctx))
		if f.DisableCleanup {
			f.Log().Logln("Skipping cleanup...")
			return
		}
		f.Log().Logln("Start landscape cleanup...")
		for ns := range dumper.namespaces {
			utils.ExpectNoError(CleanupLandscaperResources(ctx, f.Client, ns))
		}
	})
	ginkgo.BeforeEach(func() {
		dumper.ClearNamespaces()
	})
	return dumper
}

// IsRegistryEnabled returns true if a docker registry is configured.
func (f *Framework) IsRegistryEnabled() bool {
	return f.RegistryConfig != nil
}

// CleanupLandscaperResources force cleans up all landscaper resources.
func CleanupLandscaperResources(ctx context.Context, kubeClient client.Client, ns string) error {
	instList := &lsv1alpha1.InstallationList{}
	if err := kubeClient.List(ctx, instList, client.InNamespace(ns)); err != nil {
		return err
	}
	for _, obj := range instList.Items {
		if err := envtest.CleanupForObject(ctx, kubeClient, &obj, time.Minute); err != nil {
			return err
		}
	}
	execList := &lsv1alpha1.ExecutionList{}
	if err := kubeClient.List(ctx, execList, client.InNamespace(ns)); err != nil {
		return err
	}
	for _, obj := range execList.Items {
		if err := envtest.CleanupForObject(ctx, kubeClient, &obj, time.Minute); err != nil {
			return err
		}
	}
	diList := &lsv1alpha1.DeployItemList{}
	if err := kubeClient.List(ctx, diList, client.InNamespace(ns)); err != nil {
		return err
	}
	for _, obj := range diList.Items {
		if err := envtest.CleanupForObject(ctx, kubeClient, &obj, time.Minute); err != nil {
			return err
		}
	}
	cmList := &corev1.ConfigMapList{}
	if err := kubeClient.List(ctx, instList, client.InNamespace(ns)); err != nil {
		return err
	}
	for _, obj := range cmList.Items {
		if err := envtest.CleanupForObject(ctx, kubeClient, &obj, time.Second); err != nil {
			return err
		}
	}
	return nil
}
