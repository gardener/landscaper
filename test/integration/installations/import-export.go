// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func ImportExportTests(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "installations", "testdata", "test1")
	)

	Describe("Imports/Exports", func() {

		var (
			state = f.Register()
			ctx   context.Context
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		AfterEach(func() {
			ctx.Done()
		})

		It("should pass imports/exports correctly to/from subinstallations", func() {
			By("Create secrets and targets")
			// dummy secret
			secret := &k8sv1.Secret{}
			utils.ExpectNoError(utils.ReadResourceFromFile(secret, path.Join(testdataDir, "10-dummy-secret.yaml")))
			secret.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, secret))
			expectedDataExport := string(secret.Data["value"])
			// dummy target
			target := &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target, path.Join(testdataDir, "10-dummy-target.yaml")))
			target.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, target))
			expectedTargetExport := target.Spec
			// component descriptor secret
			secret2 := &k8sv1.Secret{}
			utils.ExpectNoError(utils.ReadResourceFromFile(secret2, path.Join(testdataDir, "10-cdimport-secret.yaml")))
			secret2.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, secret2))
			tmpData := secret2.Data["componentDescriptor"]
			tmpDataJSON, err := yaml.ToJSON(tmpData)
			utils.ExpectNoError(err)
			secretCD := &cdv2.ComponentDescriptor{}
			utils.ExpectNoError(json.Unmarshal(tmpDataJSON, secretCD))
			// component descriptor configmap
			cm := &k8sv1.ConfigMap{}
			utils.ExpectNoError(utils.ReadResourceFromFile(cm, path.Join(testdataDir, "10-cdimport-configmap.yaml")))
			cm.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, cm))
			tmpDataString := cm.Data["componentDescriptor"]
			tmpDataJSON, err = yaml.ToJSON([]byte(tmpDataString))
			utils.ExpectNoError(err)
			cmCD := &cdv2.ComponentDescriptor{}
			utils.ExpectNoError(json.Unmarshal(tmpDataJSON, cmCD))

			By("Create root installation")
			root := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(root, path.Join(testdataDir, "00-root-installation.yaml")))
			root.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, root))

			By("verify that subinstallation has been created")
			subinst := &lsv1alpha1.Installation{}
			Eventually(func() (bool, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(root), root)
				if err != nil || len(root.Status.InstallationReferences) == 0 {
					return false, err
				}
				err = f.Client.Get(ctx, root.Status.InstallationReferences[0].Reference.NamespacedName(), subinst)
				if err != nil {
					return false, err
				}
				return true, nil
			}, timeoutTime, resyncTime).Should(BeTrue(), "unable to fetch subinstallation")

			By("verify that installations are succeeded")
			Eventually(func() (lsv1alpha1.ComponentInstallationPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(root), root)
				if err != nil {
					return "", err
				}
				return root.Status.Phase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ComponentPhaseSucceeded), "root installation should be in phase '%s'", lsv1alpha1.ComponentPhaseSucceeded)
			Eventually(func() (lsv1alpha1.ComponentInstallationPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(subinst), subinst)
				if err != nil {
					return "", err
				}
				return subinst.Status.Phase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ComponentPhaseSucceeded), "subinstallation should be in phase '%s'", lsv1alpha1.ComponentPhaseSucceeded)

			labels := map[string]string{
				lsv1alpha1.DataObjectKeyLabel:        "dataExp",
				lsv1alpha1.DataObjectSourceTypeLabel: "export",
				lsv1alpha1.DataObjectSourceLabel:     fmt.Sprintf("Inst.%s", root.Name),
			}

			// data export
			By("verify data exports")
			rawDOExports := &lsv1alpha1.DataObjectList{}
			utils.ExpectNoError(f.Client.List(ctx, rawDOExports, client.InNamespace(state.Namespace), client.MatchingLabels(labels)))
			// remove entries which have non-empty context labels
			doExports := []lsv1alpha1.DataObject{}
			for _, elem := range rawDOExports.Items {
				con, ok := elem.Labels[lsv1alpha1.DataObjectContextLabel]
				if !ok || len(con) == 0 {
					doExports = append(doExports, elem)
				}
			}
			Expect(doExports).To(HaveLen(1), "there should be exactly one root-level dataobject export")
			Expect(doExports).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Data": WithTransform(func(aj lsv1alpha1.AnyJSON) interface{} {
					var res interface{}
					err := json.Unmarshal(aj.RawMessage, &res)
					if err != nil {
						return nil
					}
					return res
				}, BeEquivalentTo(expectedDataExport)),
			})))

			// target export
			By("verify target exports")
			labels[lsv1alpha1.DataObjectKeyLabel] = "targetExp"
			rawTargetExports := &lsv1alpha1.TargetList{}
			utils.ExpectNoError(f.Client.List(ctx, rawTargetExports, client.InNamespace(state.Namespace), client.MatchingLabels(labels)))
			targetExports := []lsv1alpha1.Target{}
			for _, elem := range rawTargetExports.Items {
				con, ok := elem.Labels[lsv1alpha1.DataObjectContextLabel]
				if !ok || len(con) == 0 {
					targetExports = append(targetExports, elem)
				}
			}
			Expect(targetExports).To(HaveLen(1), "there should be exactly one root-level target export")
			Expect(targetExports).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Spec": BeEquivalentTo(expectedTargetExport),
			})))

			// targetlist import
			// targetlists cannot be exported, so check for successful import in subinstallation instead
			By("verify targetlist imports")
			labels = map[string]string{
				lsv1alpha1.DataObjectKeyLabel:        "subTargetListImp",
				lsv1alpha1.DataObjectSourceTypeLabel: "import",
				lsv1alpha1.DataObjectSourceLabel:     fmt.Sprintf("Inst.%s", subinst.Name),
				lsv1alpha1.DataObjectContextLabel:    fmt.Sprintf("Inst.%s", subinst.Name),
			}
			tlImport := &lsv1alpha1.TargetList{}
			utils.ExpectNoError(f.Client.List(ctx, tlImport, client.InNamespace(state.Namespace), client.MatchingLabels(labels)))
			Expect(tlImport.Items).To(HaveLen(3))
			for _, elem := range tlImport.Items {
				Expect(elem).To(MatchFields(IgnoreExtras, Fields{
					"Spec": BeEquivalentTo(expectedTargetExport),
				}))
			}

			// empty targetlist import
			By("verify empty targetlist import")
			labels = map[string]string{
				lsv1alpha1.DataObjectKeyLabel:        "subEmptyTargetListImp",
				lsv1alpha1.DataObjectSourceTypeLabel: "import",
				lsv1alpha1.DataObjectSourceLabel:     fmt.Sprintf("Inst.%s", subinst.Name),
				lsv1alpha1.DataObjectContextLabel:    fmt.Sprintf("Inst.%s", subinst.Name),
			}
			tlImport = &lsv1alpha1.TargetList{}
			utils.ExpectNoError(f.Client.List(ctx, tlImport, client.InNamespace(state.Namespace), client.MatchingLabels(labels)))
			Expect(tlImport.Items).To(HaveLen(0))

			// component descriptor imports
			// exporting component descriptors is not possible and checking for in-cluster objects doesn't work either
			// therefore the imports are rendered into the deploy item config
			By("fetch deploy item provider status")
			// fetch execution
			exec := &lsv1alpha1.Execution{}
			utils.ExpectNoError(f.Client.Get(ctx, subinst.Status.ExecutionReference.NamespacedName(), exec))
			// there should be exactly one deploy item
			Expect(exec.Status.DeployItemReferences).To(HaveLen(1))
			Expect(exec.Status.DeployItemReferences[0].Name).To(BeEquivalentTo("submain-import"))
			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(f.Client.Get(ctx, exec.Status.DeployItemReferences[0].Reference.NamespacedName(), di))
			// extract provider configuration from deploy item
			conf := &mockv1alpha1.ProviderConfiguration{}
			utils.ExpectNoError(json.Unmarshal(di.Spec.Configuration.Raw, conf))
			// extract ProviderStatus field from configuration
			providerStatusDef := map[string]json.RawMessage{}
			utils.ExpectNoError(json.Unmarshal(conf.ProviderStatus.Raw, &providerStatusDef))

			By("verify component descriptor imports")
			// provider status should contain component descriptor imports
			//  secret ref
			cdImportBySecretRaw, ok := providerStatusDef["cdImportBySecretRef"]
			Expect(ok).To(BeTrue(), "cdImportBySecretRef not found in provider status definition")
			cdImportBySecret := &cdv2.ComponentDescriptor{}
			utils.ExpectNoError(json.Unmarshal(cdImportBySecretRaw, cdImportBySecret))
			Expect(cdImportBySecret).To(Equal(secretCD))
			//  configmap ref
			cdImportByConfigMapRaw, ok := providerStatusDef["cdImportByConfigMapRef"]
			Expect(ok).To(BeTrue(), "cdImportByConfigMapRef not found in provider status definition")
			cdImportByConfigMap := &cdv2.ComponentDescriptor{}
			utils.ExpectNoError(json.Unmarshal(cdImportByConfigMapRaw, cdImportByConfigMap))
			Expect(cdImportByConfigMap).To(Equal(cmCD))

			By("verify component descriptor list imports")
			//  cd list import by referencing multiple cd imports
			cdListImportByCdRefsRaw, ok := providerStatusDef["cdListImportByCdRefs"]
			Expect(ok).To(BeTrue(), "cdListImportByCdRefs not found in provider status definition")
			cdListImportByCdRefs := &cdv2.ComponentDescriptorList{}
			utils.ExpectNoError(json.Unmarshal(cdListImportByCdRefsRaw, cdListImportByCdRefs))
			Expect(cdListImportByCdRefs.Components).To(HaveLen(2))
			Expect(cdListImportByCdRefs.Components).To(ContainElement(*secretCD))
			Expect(cdListImportByCdRefs.Components).To(ContainElement(*cmCD))
			//  cd list import by referencing a cd list import
			cdListImportByListRefRaw, ok := providerStatusDef["cdListImportByListRef"]
			Expect(ok).To(BeTrue(), "cdListImportByListRef not found in provider status definition")
			cdListImportByListRef := &cdv2.ComponentDescriptorList{}
			utils.ExpectNoError(json.Unmarshal(cdListImportByListRefRaw, cdListImportByListRef))
			Expect(cdListImportByListRef.Components).To(HaveLen(2))
			Expect(cdListImportByListRef.Components).To(ContainElement(*secretCD))
			Expect(cdListImportByListRef.Components).To(ContainElement(*cmCD))
			// empty cd list import
			emptyCdListImportRaw, ok := providerStatusDef["emptyCdListImport"]
			Expect(ok).To(BeTrue(), "emptyCdListImport not found in provider status definition")
			emptyCdListImport := &cdv2.ComponentDescriptorList{}
			utils.ExpectNoError(json.Unmarshal(emptyCdListImportRaw, emptyCdListImport))
			Expect(emptyCdListImport.Components).To(HaveLen(0))
		})

	})
}
