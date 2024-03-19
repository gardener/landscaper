package targets

import (
	"context"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/matchers"
)

func TargetMapTests(ctx context.Context, f *framework.Framework) {

	Describe("Target maps", func() {

		const (
			// cluster names
			redCluster    = "red-cluster"
			blueCluster   = "blue-cluster"
			yellowCluster = "yellow-cluster"
			whiteCluster  = "white-cluster"

			// data of the deployed configmaps
			cpu       = "cpu"
			cpuRed    = "180m"
			cpuBlue   = "100m"
			cpuYellow = "140m"
		)

		var (
			testdataDir          = filepath.Join(f.RootPath, "test", "integration", "testdata", "targets", "installation-targetmap")
			pathNamespaceDO      = path.Join(testdataDir, "dataobject-namespace.yaml")
			pathConfigsDO        = path.Join(testdataDir, "dataobject-configs.yaml")
			pathUpdatedConfigsDO = path.Join(testdataDir, "dataobject-configs-upd.yaml")
			state                = f.Register()
		)

		readObject := func(ctx context.Context, obj client.Object) error {
			return state.Client.Get(ctx, client.ObjectKeyFromObject(obj), obj)
		}

		createTargets := func(ctx context.Context, names ...string) error {
			for _, name := range names {
				target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, name, f.RestConfig)
				if err != nil {
					return err
				}
				if err := state.Create(ctx, target); err != nil {
					return err
				}
			}
			return nil
		}

		deleteNamespace := func(ctx context.Context) error {
			ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: state.Namespace}}
			if err := state.Client.Get(ctx, client.ObjectKeyFromObject(ns), ns); err != nil {
				return err
			}
			if err := state.Client.Delete(ctx, ns); err != nil {
				return err
			}
			return nil
		}

		executeTest := func(installationFile1, installationFile2 string) {
			exist := matchers.Exist(state.Client)

			By("Create targets and dataobjects")
			Expect(createTargets(ctx, redCluster, blueCluster, yellowCluster, whiteCluster)).To(Succeed())
			doNamespace := &lsv1alpha1.DataObject{}
			Expect(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, doNamespace, pathNamespaceDO)).To(Succeed())
			doConfigs := &lsv1alpha1.DataObject{}
			Expect(utils.CreateDataObjectFromFile(ctx, state.State, doConfigs, pathConfigsDO)).To(Succeed())

			By("Create installation and reconcile it")
			inst := &lsv1alpha1.Installation{}
			Expect(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, installationFile1)))
			Expect(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute)).To(Succeed())

			By("Check deployed configmaps")
			cmRed := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm-red", Namespace: state.Namespace}}
			Expect(readObject(ctx, cmRed)).To(Succeed())
			Expect(cmRed.Data).To(HaveKeyWithValue(cpu, cpuRed))
			cmBlue := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm-blue", Namespace: state.Namespace}}
			Expect(readObject(ctx, cmBlue)).To(Succeed())
			Expect(cmBlue.Data).To(HaveKeyWithValue(cpu, cpuBlue))
			cmYellow := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm-yellow", Namespace: state.Namespace}}
			Expect(cmYellow).NotTo(exist)

			By("Update dataobject")
			Expect(utils.UpdateDataObjectFromFile(ctx, state.State, doConfigs, pathUpdatedConfigsDO))

			By("Update installation and reconcile it")
			Expect(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, installationFile2)))
			Expect(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute)).To(Succeed())

			By("Check deployed configmaps")
			// Before the update, there were two keys: red and blue. The update removed red and added yellow.
			// We check that the blue and yellow configmap exist and have the correct content.
			// Moreover, we check that the blue configmap is just updated, rather than deleted and recreated.
			Expect(cmRed).NotTo(exist)
			cmBlue2 := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm-blue", Namespace: state.Namespace}}
			Expect(readObject(ctx, cmBlue2)).To(Succeed())
			Expect(cmBlue2.UID).To(Equal(cmBlue.UID))
			Expect(readObject(ctx, cmYellow)).To(Succeed())
			Expect(cmYellow.Data).To(HaveKeyWithValue(cpu, cpuYellow))

			By("Delete installation")
			Expect(state.Client.Delete(ctx, inst)).To(Succeed())
			Expect(lsutils.WaitForInstallationToBeDeleted(ctx, f.Client, inst, 2*time.Minute)).To(Succeed())
			Expect(cmBlue).NotTo(exist)
			Expect(cmYellow).NotTo(exist)

			By("Cleanup namespace")
			Expect(deleteNamespace(ctx)).To(Succeed())
		}

		It("should create a deployitem per target", func() {
			executeTest("installation-01.yaml", "installation-01-upd.yaml")
		})

		It("should pass a targetmap reference and create a deployitem per target", func() {
			executeTest("installation-02.yaml", "installation-02-upd.yaml")
		})

		It("should twice pass a targetmap reference and create a deployitem per target", func() {
			executeTest("installation-03.yaml", "installation-03-upd.yaml")
		})

		It("should create a sub-installation per target", func() {
			executeTest("installation-04.yaml", "installation-04.yaml")
		})

		It("should pass a targetmap reference and create a sub-installation per target", func() {
			executeTest("installation-05.yaml", "installation-05.yaml")
		})

		It("should twice pass a targetmap reference and create a sub-installation per target", func() {
			executeTest("installation-06.yaml", "installation-06.yaml")
		})

		It("should compose a targetmap", func() {
			executeTest("installation-07.yaml", "installation-07-upd.yaml")
		})

		It("should pass targets and compose a targetmap", func() {
			executeTest("installation-08.yaml", "installation-08-upd.yaml")
		})

		It("should compose a targetmap from exports", func() {
			executeTest("installation-09.yaml", "installation-09.yaml")
		})

		It("should pass a target and compose a targetmap from exports", func() {
			executeTest("installation-10.yaml", "installation-10.yaml")
		})

	})
}
