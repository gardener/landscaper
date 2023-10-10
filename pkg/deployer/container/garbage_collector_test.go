// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	lsutils "github.com/gardener/landscaper/pkg/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	errors2 "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/apis/deployer/container"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/utils/simplelogger"
	testutils "github.com/gardener/landscaper/test/utils"

	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	containerctlr "github.com/gardener/landscaper/pkg/deployer/container"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("GarbageCollector", func() {

	var (
		lsState   *envtest.State
		hostState *envtest.State
		lsMgr     manager.Manager
		hostMgr   manager.Manager
		gc        *containerctlr.GarbageCollector
		ctx       context.Context
		cancel    context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		logger := logging.Wrap(simplelogger.WithTimestamps(simplelogger.NewIOLogger(GinkgoWriter)))
		var err error
		lsMgr, err = manager.New(testenv.Env.Config, manager.Options{
			Scheme:             api.LandscaperScheme,
			MetricsBindAddress: "0",
			Logger:             logger.WithName("lsManager").Logr(),
			NewClient:          lsutils.NewUncachedClient(lsutils.LsResourceClientBurstDefault, lsutils.LsResourceClientQpsDefault),
		})
		Expect(err).ToNot(HaveOccurred())

		hostMgr, err = manager.New(hostTestEnv.Env.Config, manager.Options{
			Scheme:             scheme.Scheme,
			MetricsBindAddress: "0",
			Logger:             logger.WithName("hostManager").Logr(),
			NewClient:          lsutils.NewUncachedClient(lsutils.LsResourceClientBurstDefault, lsutils.LsResourceClientQpsDefault),
		})
		Expect(err).ToNot(HaveOccurred())

		hostState, err = hostTestEnv.InitState(ctx)
		Expect(err).ToNot(HaveOccurred())
		lsState, err = testenv.InitState(ctx)
		Expect(err).ToNot(HaveOccurred())

		Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, lsState.Namespace)).To(Succeed())

		gc = containerctlr.NewGarbageCollector(logger, testenv.Client, hostTestEnv.Client, "test", hostState.Namespace, containerv1alpha1.GarbageCollection{
			Worker:             1,
			RequeueTimeSeconds: 1,
		})
		Expect(gc.Add(hostMgr, false)).To(Succeed())

		Expect(testutils.AddMimicKCMSecretControllerToManager(hostMgr)).To(Succeed())

		go func() {
			Expect(lsMgr.Start(ctx)).To(Succeed())
		}()
		Expect(lsMgr.GetCache().WaitForCacheSync(ctx)).To(BeTrue())
		go func() {
			Expect(hostMgr.Start(ctx)).To(Succeed())
		}()
		Expect(hostMgr.GetCache().WaitForCacheSync(ctx)).To(BeTrue())
	})

	AfterEach(func() {
		defer cancel()
		Expect(lsState.CleanupState(ctx, envtest.WaitForDeletion(true), envtest.WithRestConfig(testenv.Env.Config))).To(Succeed())
		Expect(hostState.CleanupState(ctx, envtest.WaitForDeletion(true), envtest.WithRestConfig(hostTestEnv.Env.Config))).To(Succeed())
	})

	Context("RBAC resources", func() {
		It("should garbage collect service accounts that have no corresponding deployitem", func() {
			di := &lsv1alpha1.DeployItem{}
			di.Name = "not"
			di.Namespace = lsState.Namespace

			sa := &corev1.ServiceAccount{}
			sa.Name = containerctlr.InitContainerServiceAccountName(di)
			sa.Namespace = hostState.Namespace
			containerctlr.InjectDefaultLabels(sa, containerctlr.DefaultLabels("test", "a", di.Name, di.Namespace))
			Expect(hostState.Create(ctx, sa)).To(Succeed())

			Eventually(func() error {
				err := hostTestEnv.Client.Get(ctx, kutil.ObjectKeyFromObject(sa), &corev1.ServiceAccount{})
				if err != nil {
					if apierrors.IsNotFound(err) {
						return nil
					}
					return err
				}
				return errors.New("still exists")
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "service account should be deleted")
		})

		It("should garbage collect all rbac related resources", func() {
			ctx := context.Background()
			di := &lsv1alpha1.DeployItem{}
			di.Name = "not"
			di.Namespace = lsState.Namespace
			defaultLabels := containerctlr.DefaultLabels("test", "a", di.Name, di.Namespace)

			Expect(lsState.Create(ctx, di)).To(Succeed())

			ensureSAResult, err := containerctlr.EnsureServiceAccounts(ctx, hostTestEnv.Client, di, hostState.Namespace, defaultLabels)
			Expect(err).ToNot(HaveOccurred())

			initSA := &corev1.ServiceAccount{}
			initSA.Name = containerctlr.InitContainerServiceAccountName(di)
			initSA.Namespace = hostState.Namespace
			initRole := &rbacv1.Role{}
			initRole.Name = initSA.Name
			initRole.Namespace = initSA.Namespace
			initRolebinding := &rbacv1.RoleBinding{}
			initRolebinding.Name = initSA.Name
			initRolebinding.Namespace = initSA.Namespace
			initSecret := &corev1.Secret{}
			initSecret.Name = ensureSAResult.InitContainerServiceAccountSecret.Name
			initSecret.Namespace = ensureSAResult.InitContainerServiceAccountSecret.Namespace

			waitSA := &corev1.ServiceAccount{}
			waitSA.Name = containerctlr.WaitContainerServiceAccountName(di)
			waitSA.Namespace = hostState.Namespace
			waitRole := &rbacv1.Role{}
			waitRole.Name = waitSA.Name
			waitRole.Namespace = waitSA.Namespace
			waitRolebinding := &rbacv1.RoleBinding{}
			waitRolebinding.Name = waitSA.Name
			waitRolebinding.Namespace = waitSA.Namespace
			waitSecret := &corev1.Secret{}
			waitSecret.Name = ensureSAResult.WaitContainerServiceAccountSecret.Name
			waitSecret.Namespace = ensureSAResult.WaitContainerServiceAccountSecret.Namespace

			resources := map[string]client.Object{
				"initServiceAccount": initSA,
				"initRole":           initRole,
				"initRoleBinding":    initRolebinding,
				"initTokenSecret":    initSecret,
				"waitServiceAccount": waitSA,
				"waitRole":           waitRole,
				"waitRoleBinding":    waitRolebinding,
				"waitTokenSecret":    waitSecret,
			}

			Expect(testenv.Client.Delete(ctx, di)).To(Succeed())
			Eventually(func() error {
				if err := testenv.Client.Get(ctx, client.ObjectKeyFromObject(di), di); err != nil {
					if apierrors.IsNotFound(err) {
						return nil
					}
				}
				return fmt.Errorf("deployitem %s still exits", client.ObjectKeyFromObject(di).String())
			}, 10*time.Second, 1*time.Second).Should(Succeed())
			Eventually(func() error {
				var allErrs []error
				for name, obj := range resources {
					err := hostTestEnv.Client.Get(ctx, kutil.ObjectKeyFromObject(obj), obj.DeepCopyObject().(client.Object))
					if err != nil {
						if apierrors.IsNotFound(err) {
							return nil
						}
						allErrs = append(allErrs, err)
						continue
					}
					allErrs = append(allErrs, fmt.Errorf("%s still exists", name))
				}
				if len(allErrs) != 0 {
					return errors2.NewAggregate(allErrs)
				}
				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "service accounts should be deleted")
		})

		It("should not garbage collect a service accounts that is not managed by the deployer", func() {
			di := &lsv1alpha1.DeployItem{}
			di.Name = "not"
			di.Namespace = lsState.Namespace

			sa := &corev1.ServiceAccount{}
			sa.Name = containerctlr.InitContainerServiceAccountName(di)
			sa.Namespace = hostState.Namespace
			Expect(hostState.Create(ctx, sa)).To(Succeed())

			time.Sleep(10 * time.Second)
			Expect(hostTestEnv.Client.Get(ctx, kutil.ObjectKeyFromObject(sa), &corev1.ServiceAccount{})).To(Succeed())
		})

		It("should not garbage collect a service accounts with a deployitem", func() {
			di := &lsv1alpha1.DeployItem{}
			di.Name = "not"
			di.Namespace = lsState.Namespace
			Expect(lsState.Create(ctx, di)).To(Succeed())

			sa := &corev1.ServiceAccount{}
			sa.Name = containerctlr.InitContainerServiceAccountName(di)
			sa.Namespace = hostState.Namespace
			containerctlr.InjectDefaultLabels(sa, containerctlr.DefaultLabels("test", "a", di.Name, di.Namespace))
			Expect(hostState.Create(ctx, sa)).To(Succeed())

			time.Sleep(10 * time.Second)
			Expect(hostTestEnv.Client.Get(ctx, kutil.ObjectKeyFromObject(sa), &corev1.ServiceAccount{})).To(Succeed())
		})
	})

	Context("Secrets", func() {
		It("should garbage collect secrets that have no corresponding deployitem", func() {
			di := &lsv1alpha1.DeployItem{}
			di.Name = "not"
			di.Namespace = lsState.Namespace

			secret := &corev1.Secret{}
			secret.Name = "test"
			secret.Namespace = hostState.Namespace
			containerctlr.InjectDefaultLabels(secret, containerctlr.DefaultLabels("test", "a", di.Name, di.Namespace))
			Expect(hostState.Create(ctx, secret)).To(Succeed())

			Eventually(func() error {
				err := hostTestEnv.Client.Get(ctx, kutil.ObjectKeyFromObject(secret), &corev1.Secret{})
				if err != nil {
					if apierrors.IsNotFound(err) {
						return nil
					}
					return err
				}
				return errors.New("still exists")
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "service account should be deleted")
		})

		It("should not garbage collect a secret that is not managed by the deployer", func() {
			secret := &corev1.Secret{}
			secret.Name = "test"
			secret.Namespace = hostState.Namespace
			Expect(hostState.Create(ctx, secret)).To(Succeed())

			time.Sleep(10 * time.Second)
			Expect(hostTestEnv.Client.Get(ctx, kutil.ObjectKeyFromObject(secret), &corev1.Secret{})).To(Succeed())
		})

		It("should not garbage collect a secret with a deployitem", func() {
			di := &lsv1alpha1.DeployItem{}
			di.Name = "not"
			di.Namespace = lsState.Namespace
			Expect(lsState.Create(ctx, di)).To(Succeed())

			secret := &corev1.Secret{}
			secret.Name = "test"
			secret.Namespace = hostState.Namespace
			containerctlr.InjectDefaultLabels(secret, containerctlr.DefaultLabels("test", "a", di.Name, di.Namespace))
			Expect(hostState.Create(ctx, secret)).To(Succeed())

			time.Sleep(10 * time.Second)
			Expect(hostTestEnv.Client.Get(ctx, kutil.ObjectKeyFromObject(secret), &corev1.Secret{})).To(Succeed())
		})
	})

	Context("Pods", func() {

		var defaultPod = func(namespace, name string) *corev1.Pod {
			pod := &corev1.Pod{}
			pod.Name = name
			pod.Namespace = namespace
			pod.Spec.Containers = []corev1.Container{
				{
					Name:  "test",
					Image: "ubuntu",
				},
			}
			return pod
		}

		It("should garbage collect all pods that have no corresponding deployitem and are not the latest", func() {
			di := &lsv1alpha1.DeployItem{}
			di.Name = "not"
			di.Namespace = lsState.Namespace
			Expect(lsState.Create(ctx, di)).To(Succeed())

			pod := defaultPod(hostState.Namespace, "test")
			pod.Finalizers = []string{container.ContainerDeployerFinalizer}
			containerctlr.InjectDefaultLabels(pod, containerctlr.DefaultLabels("test", "a", di.Name, di.Namespace))
			Expect(hostState.Create(ctx, pod)).To(Succeed())
			pod.Status.Phase = corev1.PodPending
			Expect(hostTestEnv.Client.Status().Update(ctx, pod))
			time.Sleep(1 * time.Second) // we need to get a different creation time for pod 2

			pod2 := defaultPod(hostState.Namespace, "test2")
			pod2.Finalizers = []string{container.ContainerDeployerFinalizer}
			containerctlr.InjectDefaultLabels(pod2, containerctlr.DefaultLabels("test", "a", di.Name, di.Namespace))
			Expect(hostState.Create(ctx, pod2)).To(Succeed())
			pod2.Status.Phase = corev1.PodPending
			Expect(hostTestEnv.Client.Status().Update(ctx, pod2))

			// retrigger a reconcile for both pods
			pod.Status.Phase = corev1.PodSucceeded
			Expect(hostTestEnv.Client.Status().Update(ctx, pod))
			pod2.Status.Phase = corev1.PodSucceeded
			Expect(hostTestEnv.Client.Status().Update(ctx, pod2))

			Eventually(func() error {
				err := hostTestEnv.Client.Get(ctx, kutil.ObjectKeyFromObject(pod), &corev1.Pod{})
				if err != nil {
					if apierrors.IsNotFound(err) {
						return nil
					}
					return err
				}
				return errors.New("still exists")
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "pod should be deleted")
			Expect(hostTestEnv.Client.Get(ctx, kutil.ObjectKeyFromObject(pod2), &corev1.Pod{})).To(Succeed())
		})

		It("should not garbage collect a pod that is not managed by the deployer", func() {
			pod := defaultPod(hostState.Namespace, "test")
			Expect(hostState.Create(ctx, pod)).To(Succeed())
			pod.Status.Phase = corev1.PodSucceeded
			Expect(hostTestEnv.Client.Status().Update(ctx, pod))

			time.Sleep(10 * time.Second)
			Expect(hostTestEnv.Client.Get(ctx, kutil.ObjectKeyFromObject(pod), &corev1.Pod{})).To(Succeed())
		})

		It("should not garbage collect a pod with a deployitem", func() {
			di := &lsv1alpha1.DeployItem{}
			di.Name = "not"
			di.Namespace = lsState.Namespace
			Expect(lsState.Create(ctx, di)).To(Succeed())

			pod := defaultPod(hostState.Namespace, "test")
			pod.Finalizers = []string{container.ContainerDeployerFinalizer}
			containerctlr.InjectDefaultLabels(pod, containerctlr.DefaultLabels("test", "a", di.Name, di.Namespace))
			Expect(hostState.Create(ctx, pod)).To(Succeed())
			pod.Status.Phase = corev1.PodSucceeded
			Expect(hostTestEnv.Client.Status().Update(ctx, pod))

			time.Sleep(10 * time.Second)
			Expect(hostTestEnv.Client.Get(ctx, kutil.ObjectKeyFromObject(pod), &corev1.Pod{})).To(Succeed())
		})

		It("should not garbage collect a pod that is still running", func() {
			pod := defaultPod(hostState.Namespace, "test")
			Expect(hostState.Create(ctx, pod)).To(Succeed())
			pod.Status.Phase = corev1.PodRunning
			Expect(hostTestEnv.Client.Status().Update(ctx, pod))

			time.Sleep(10 * time.Second)
			Expect(hostTestEnv.Client.Get(ctx, kutil.ObjectKeyFromObject(pod), &corev1.Pod{})).To(Succeed())
		})

		It("should not garbage collect the latest pod", func() {
			di := &lsv1alpha1.DeployItem{}
			di.Name = "not"
			di.Namespace = lsState.Namespace
			Expect(lsState.Create(ctx, di)).To(Succeed())

			pod := defaultPod(hostState.Namespace, "test")
			pod.Finalizers = []string{container.ContainerDeployerFinalizer}
			containerctlr.InjectDefaultLabels(pod, containerctlr.DefaultLabels("test", "a", di.Name, di.Namespace))
			Expect(hostState.Create(ctx, pod)).To(Succeed())
			pod.Status.Phase = corev1.PodSucceeded
			Expect(hostTestEnv.Client.Status().Update(ctx, pod))

			time.Sleep(10 * time.Second)
			Expect(hostTestEnv.Client.Get(ctx, kutil.ObjectKeyFromObject(pod), &corev1.Pod{})).To(Succeed())
		})

		It("should garbage collect the latest pod if no finalizer is set", func() {
			di := &lsv1alpha1.DeployItem{}
			di.Name = "not"
			di.Namespace = lsState.Namespace
			Expect(lsState.Create(ctx, di)).To(Succeed())

			pod := defaultPod(hostState.Namespace, "test")
			containerctlr.InjectDefaultLabels(pod, containerctlr.DefaultLabels("test", "a", di.Name, di.Namespace))
			Expect(hostState.Create(ctx, pod)).To(Succeed())
			pod.Status.Phase = corev1.PodSucceeded
			Expect(hostTestEnv.Client.Status().Update(ctx, pod))

			time.Sleep(10 * time.Second)
			Expect(hostTestEnv.Client.Get(ctx, kutil.ObjectKeyFromObject(pod), &corev1.Pod{})).ToNot(Succeed())
		})
	})
})
