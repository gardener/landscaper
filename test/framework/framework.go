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
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	utils2 "github.com/gardener/landscaper/hack/testcluster/pkg/utils"
	lsscheme "github.com/gardener/landscaper/pkg/api"
	utils3 "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var (
	timeoutTime = 30 * time.Second
)

// OpenSourceRepositoryContext is the base url of the repository context for the gardener open source components.
// There all landscaper blueprints/components are available.
const OpenSourceRepositoryContext = "europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper"

type Options struct {
	fs                             *flag.FlagSet
	KubeconfigPath                 string
	RootPath                       string
	LsNamespace                    string
	LsVersion                      string
	DockerConfigPath               string
	DisableCleanup                 bool
	RunOnShoot                     bool
	DisableCleanupBefore           bool
	SkipWaitingForSystemComponents bool
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
	fs.BoolVar(&o.RunOnShoot, "ls-run-on-shoot", false, "runs on a shoot and not a k3s cluster")
	fs.BoolVar(&o.DisableCleanupBefore, "ls-disable-cleanup-before", false, "disables cleanup of all namespaces with prefix `test` before the tests are started")
	fs.BoolVar(&o.SkipWaitingForSystemComponents, "skip-waiting-for-system-components", false, "disables checking whether landscaper and the deployers are running in the cluster")
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
	logger utils2.Logger
	// RootPath is the filepath to the root of the landscaper repository
	RootPath string
	// RestConfig is the kubernetes rest config for the test cluster
	RestConfig *rest.Config
	// Client is the kubernetes client to interact with the test cluster
	Client       client.Client
	TargetClient client.Client
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
	// RunOnShoot tests are executed on shoot and not a k3s cluster (only for compatibility with old setup)
	RunOnShoot bool

	// RegistryConfig defines the oci registry config file.
	// It is expected that the configfile contains exactly one server.
	RegistryConfig *configfile.ConfigFile

	RegistryConfigPath string

	RegistryCAPath string
	// RegistryBasePath defines the base path for the configured registry.
	// The base path is used to construct references for artifacts.
	RegistryBasePath string
	// OCIClient is a oci client that can up and download artifacts from the configured registry
	OCIClient ociclient.Client
	// OCICache is the oci store of the local oci client
	OCICache cache.Cache

	TestsFailed bool
}

func New(logger utils2.Logger, cfg *Options) (*Framework, error) {
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
		RunOnShoot:     cfg.RunOnShoot,
		TestsFailed:    false,
	}

	var err error
	f.RestConfig, err = clientcmd.BuildConfigFromFlags("", cfg.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to parse kubeconfig: %w", err)
	}
	innerClient, err := utils3.NewUncached(utils3.LsResourceClientBurstDefault, utils3.LsResourceClientQpsDefault, f.RestConfig,
		client.Options{Scheme: lsscheme.LandscaperScheme})

	if err != nil {
		return nil, fmt.Errorf("unable to build kubernetes client: %w", err)
	}

	f.Client = envtest.NewRetryingClient(innerClient, logger)

	f.ClientSet, err = utils3.NewForConfig(utils3.LsResourceClientBurstDefault, utils3.LsResourceClientQpsDefault, f.RestConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to build kubernetes clientset: %w", err)
	}

	if len(cfg.DockerConfigPath) != 0 {
		f.RegistryConfigPath = cfg.DockerConfigPath
		f.RegistryCAPath = filepath.Join(filepath.Dir(cfg.DockerConfigPath), "cacerts.crt")

		data, err := os.ReadFile(cfg.DockerConfigPath)
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

		ociKeyring, err := credentials.NewBuilder(logging.Discard().Logr()).FromConfigFiles(cfg.DockerConfigPath).Build()
		if err != nil {
			return nil, fmt.Errorf("unable to build oci keyring: %w", err)
		}
		f.OCICache, err = cache.NewCache(logging.Discard().Logr())
		if err != nil {
			return nil, err
		}

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		httpClient := http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}
		f.OCIClient, err = ociclient.NewClient(logging.Discard().Logr(),
			ociclient.WithKeyring(ociKeyring),
			ociclient.WithCache(f.OCICache),
			ociclient.WithHTTPClient(httpClient))
		if err != nil {
			return nil, fmt.Errorf("unable to build oci client: %w", err)
		}
	}
	return f, nil
}

// Log returns the default logger
func (f *Framework) Log() utils2.Logger {
	return f.logger
}

// TestLog returns a new testlogger that logs to the ginkgo managed writer
func (f *Framework) TestLog() utils2.Logger {
	return utils2.NewLoggerFromWriter(ginkgo.GinkgoWriter)
}

// WaitForSystemComponents waits for all system component of the landscaper to be ready
func (f *Framework) WaitForSystemComponents(ctx context.Context) error {
	if len(f.LsNamespace) == 0 {
		return nil
	}
	f.logger.WithTimestamp().Logfln("Waiting for Landscaper components to be ready in %s", f.LsNamespace)
	// get all deployments
	deploymentNames := []string{"landscaper", "landscaper-webhooks", "container-deployer", "helm-deployer", "manifest-deployer", "mock-deployer"}

	for _, deploymentName := range deploymentNames {
		if err := utils.WaitForDeploymentToBeReady(ctx, f.Log(), f.Client, client.ObjectKey{Namespace: f.LsNamespace, Name: deploymentName}, 10*time.Minute); err != nil {
			return err
		}
	}

	return nil
}

// NewState creates a new state with a test namespace.
// It also returns a cleanup function that should be called when the test has finished.
func (f *Framework) NewState(ctx context.Context) (*envtest.State, CleanupFunc, error) {
	state, err := envtest.InitStateWithNamespace(ctx, f.Client, f.logger, false)
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
		return state.CleanupState(ctx, envtest.WithCleanupTimeout(time.Minute))
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
func (f *Framework) Register() *State {
	state := &State{}

	ginkgo.BeforeEach(func() {
		ctx := context.Background()
		defer ctx.Done()
		envState, cleanup, err := f.NewState(ctx)
		utils.ExpectNoError(err)
		dumper := NewDumper(f.logger, f.Client, f.ClientSet, f.LsNamespace, envState.Namespace)
		dumper.startTime = time.Now()

		s := State{
			State:   envState,
			dumper:  dumper,
			cleanup: cleanup,
		}
		*state = s

		err = f.prepareNextTest(ctx, state.Namespace)
		utils.ExpectNoError(err)
	})

	ginkgo.AfterEach(func() {
		f.TestsFailed = f.TestsFailed || ginkgo.CurrentSpecReport().Failed()
		ctx := context.Background()
		defer ctx.Done()
		dumper := state.dumper
		dumper.endTime = time.Now()

		// dump before cleanup if the test failed
		f.Log().Logln("Check if test failed...")
		//if ginkgo.CurrentSpecReport().Failed() {
		//	utils.ExpectNoError(dumper.Dump(ctx))
		//}

		if !ginkgo.CurrentSpecReport().Failed() {
			if err := state.cleanup(ctx); err != nil {
				{
					// try to dump
					//if err := dumper.Dump(ctx); err != nil {
					//	f.logger.Logln(err.Error())
					//}
					utils.ExpectNoError(err)
				}
			}
		}
	})
	return state
}

// IsRegistryEnabled returns true if a docker registry is configured.
func (f *Framework) IsRegistryEnabled() bool {
	return f.RegistryConfig != nil
}

func (f *Framework) CleanupBeforeTestNamespaces(ctx context.Context) error {
	testNamespaces := &corev1.NamespaceList{}
	if err := f.Client.List(ctx, testNamespaces); err != nil {
		return err
	}

	for _, nextNamespace := range testNamespaces.Items {
		if strings.HasPrefix(nextNamespace.Name, "test") && len(nextNamespace.Name) > len("test") {
			if err := f.cleanupBeforeObjectsInTestNamespace(ctx, nextNamespace); err != nil {
				return err
			}
		}
	}

	if err := f.Client.List(ctx, testNamespaces); err != nil {
		return err
	}

	for _, nextNamespace := range testNamespaces.Items {
		if strings.HasPrefix(nextNamespace.Name, "test") && len(nextNamespace.Name) > len("test") {
			if nextNamespace.GetDeletionTimestamp().IsZero() {
				if err := f.Client.Delete(ctx, &nextNamespace); err != nil && !apierrors.IsNotFound(err) {
					return err
				}
			}
		}
	}

	return nil
}

func (f *Framework) cleanupBeforeObjectsInTestNamespace(ctx context.Context, namespace corev1.Namespace) error {
	instList := &lsv1alpha1.InstallationList{}
	if err := f.Client.List(ctx, instList, client.InNamespace(namespace.Name)); err != nil {
		return err
	}
	for _, obj := range instList.Items {
		if err := envtest.CleanupForObject(ctx, f.logger, f.Client, &obj, time.Second); err != nil {
			return err
		}
	}

	execList := &lsv1alpha1.ExecutionList{}
	if err := f.Client.List(ctx, execList, client.InNamespace(namespace.Name)); err != nil {
		return err
	}
	for _, obj := range execList.Items {
		if err := envtest.CleanupForObject(ctx, f.logger, f.Client, &obj, time.Second); err != nil {
			return err
		}
	}

	diList := &lsv1alpha1.DeployItemList{}
	if err := f.Client.List(ctx, diList, client.InNamespace(namespace.Name)); err != nil {
		return err
	}
	for _, obj := range diList.Items {
		if err := envtest.CleanupForObject(ctx, f.logger, f.Client, &obj, time.Second); err != nil {
			return err
		}
	}

	podList := &corev1.PodList{}
	if err := f.Client.List(ctx, podList, client.InNamespace(namespace.Name)); err != nil {
		return err
	}
	for _, obj := range podList.Items {
		if err := envtest.CleanupForObject(ctx, f.logger, f.Client, &obj, time.Second); err != nil {
			return err
		}
	}

	return nil
}

func (f *Framework) prepareNextTest(ctx context.Context, namespace string) error {
	f.Log().Logln("prepare next test")

	err := utils.WaitForContextToBeReady(ctx, f.Log(), f.Client, client.ObjectKey{Namespace: namespace, Name: "default"}, timeoutTime)
	if err != nil {
		return err
	}

	f.Log().Logln("check for ingress class")
	ingressClass := networkingv1.IngressClass{}
	err = f.Client.Get(ctx, client.ObjectKey{Name: "nginx"}, &ingressClass)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else {
		f.Log().Logln("Delete ingressClass")
		err = f.Client.Delete(ctx, &ingressClass)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	f.Log().Logln("Cleanup ValidatingWebhookConfigurations")
	hookList := &v1.ValidatingWebhookConfigurationList{}
	if err := f.Client.List(ctx, hookList); err != nil {
		return err
	}

	for i := range hookList.Items {
		hook := &hookList.Items[i]
		ann := hook.GetAnnotations()
		if len(ann) > 0 {
			releaseNamespace, ok := ann["meta.helm.sh/release-namespace"]
			if ok && strings.HasPrefix(releaseNamespace, "tests-") {
				f.Log().Logfln("Delete ValidatingWebhookConfiguration %s", hook.Name)
				if err := f.Client.Delete(ctx, hook); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
