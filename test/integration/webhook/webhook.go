// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	hdv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"

	yaml "gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RegisterTests registers all tests of the package
func RegisterTests(f *framework.Framework) {
	WebhookTest(f)
}

func WebhookTest(f *framework.Framework) {
	_ = Describe("WebhookTest", func() {
		var (
			ctx   context.Context
			state = f.Register()
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		AfterEach(func() {
			ctx.Done()
		})

		It("should have created a ValidatingWebhookConfiguration", func() {
			vwc := admissionregistrationv1.ValidatingWebhookConfiguration{}
			utils.ExpectNoError(f.Client.Get(ctx, kutil.ObjectKey("landscaper-validation-webhook", ""), &vwc))
		})

		It("should block invalid Installation resources", func() {
			// reuse installation from import/export test, because it declares lots of imports/exports, which is useful for testing the validation
			testdataDir := filepath.Join(f.RootPath, "test", "integration", "installations", "testdata", "test1")

			// walk over test files and create them
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(filepath.WalkDir(testdataDir, func(path string, d fs.DirEntry, err error) error {
				if path == testdataDir {
					return nil
				}
				if d.IsDir() {
					return fs.SkipDir
				}
				if d.Name() == "00-root-installation.yaml" {
					utils.ExpectNoError(utils.ReadResourceFromFile(inst, path))
					inst.SetNamespace(state.Namespace)
					lsv1alpha1helper.SetOperation(&inst.ObjectMeta, lsv1alpha1.ReconcileOperation)
					metav1.SetMetaDataAnnotation(&inst.ObjectMeta, lsv1alpha1.DeleteWithoutUninstallAnnotation, "true")
				} else {
					// parse file and read kind
					data, err := ioutil.ReadFile(path)
					utils.ExpectNoError(err)
					parsed := map[string]interface{}{}
					utils.ExpectNoError(yaml.Unmarshal(data, parsed))
					kind, ok := parsed["kind"].(string)
					Expect(ok).To(BeTrue())
					var obj client.Object
					switch kind {
					case "ConfigMap":
						obj = &corev1.ConfigMap{}
					case "Secret":
						obj = &corev1.Secret{}
					case "Target":
						obj = &lsv1alpha1.Target{}
					case "Installation":
						obj = &lsv1alpha1.Installation{}
					default:
						Fail(fmt.Sprintf("manifest of unknown kind '%s' in test folder, probably this test needs to be expanded", kind))
					}
					utils.ExpectNoError(utils.ReadResourceFromFile(obj, path))
					obj.SetNamespace(state.Namespace)
					utils.ExpectNoError(state.Create(ctx, obj))
				}
				return nil
			}))
			Expect(inst.Name).ToNot(BeEmpty()) // root installation should have been found

			// apply root installation into cluster and wait for it to be succeeded
			lsv1alpha1helper.SetOperation(&inst.ObjectMeta, lsv1alpha1.ReconcileOperation)
			utils.ExpectNoError(state.Create(ctx, inst))
			Eventually(func() lsv1alpha1.InstallationPhase {
				utils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
				return inst.Status.InstallationPhase
			}, 30*time.Second, 1*time.Second).Should(Equal(lsv1alpha1.InstallationPhaseSucceeded))

			// make installation invalid by duplicating the first export
			invalidInst := inst.DeepCopy()
			invalidInst.Spec.Exports.Data = append(invalidInst.Spec.Exports.Data, invalidInst.Spec.Exports.Data[0])
			err := state.Client.Update(ctx, invalidInst)
			Expect(err).To(HaveOccurred()) // validation webhook should have denied this
			Expect(err.Error()).To(HavePrefix("admission webhook \"installations.validation.landscaper.gardener.cloud\" denied the request"))

			// create invalid installation by creating different installation with same exports
			invalidInst = inst.DeepCopy()
			invalidInst.SetName("root2")
			err = state.Create(ctx, invalidInst)
			Expect(err).To(HaveOccurred()) // validation webhook should have denied this
			Expect(err.Error()).To(HavePrefix("admission webhook \"installations.validation.landscaper.gardener.cloud\" denied the request"))
			// the error should contain the conflicting export names
			for _, exp := range inst.Spec.Exports.Data {
				Expect(err.Error()).To(ContainSubstring(exp.DataRef))
			}
			for _, exp := range inst.Spec.Exports.Targets {
				Expect(err.Error()).To(ContainSubstring(exp.Target))
			}
			for exp := range inst.Spec.ExportDataMappings {
				Expect(err.Error()).To(ContainSubstring(exp))
			}

		})

		It("should block invalid Execution resources", func() {
			conf := hdv1alpha1.Configuration{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ProviderConfiguration",
					APIVersion: "helm.deployer.landscaper.gardener.cloud/v1alpha1",
				},
				OCI: &config.OCIConfiguration{
					AllowPlainHttp: false,
				},
				TargetSelector: []lsv1alpha1.TargetSelector{},
			}
			obj := conf.DeepCopyObject()
			raw := runtime.RawExtension{}
			utils.ExpectNoError(runtime.Convert_runtime_Object_To_runtime_RawExtension(&obj, &raw, nil))

			// create invalid execution (cyclic dependencies)
			exec := &lsv1alpha1.Execution{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-execution",
					Namespace: state.Namespace,
				},
				Spec: lsv1alpha1.ExecutionSpec{
					DeployItems: lsv1alpha1.DeployItemTemplateList{
						{
							Name:          "a",
							Type:          "landscaper.gardener.cloud/helm",
							DependsOn:     []string{"b"},
							Configuration: &raw,
						},
						{
							Name:          "b",
							Type:          "landscaper.gardener.cloud/helm",
							DependsOn:     []string{"a"},
							Configuration: &raw,
						},
					},
				},
			}

			err := state.Create(ctx, exec)
			Expect(err).To(HaveOccurred()) // validation webhook should have denied this
			Expect(err.Error()).To(HavePrefix("admission webhook \"executions.validation.landscaper.gardener.cloud\" denied the request"))
		})

		It("should block invalid DeployItem resources", func() {
			conf := hdv1alpha1.Configuration{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ProviderConfiguration",
					APIVersion: "helm.deployer.landscaper.gardener.cloud/v1alpha1",
				},
				OCI: &config.OCIConfiguration{
					AllowPlainHttp: false,
				},
				TargetSelector: []lsv1alpha1.TargetSelector{},
			}
			obj := conf.DeepCopyObject()
			raw := runtime.RawExtension{}
			utils.ExpectNoError(runtime.Convert_runtime_Object_To_runtime_RawExtension(&obj, &raw, nil))

			di := &lsv1alpha1.DeployItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployitem",
					Namespace: state.Namespace,
				},
				Spec: lsv1alpha1.DeployItemSpec{
					Type: "landscaper.gardener.cloud/helm",
					Target: &lsv1alpha1.ObjectReference{
						Name:      "", // invalid
						Namespace: state.Namespace,
					},
					Configuration: &raw,
				},
			}

			err := state.Create(ctx, di)
			Expect(err).To(HaveOccurred()) // validation webhook should have denied this
			Expect(err.Error()).To(HavePrefix("admission webhook \"deployitems.validation.landscaper.gardener.cloud\" denied the request"))
		})

		It("should block a DeployItem type update", func() {
			di := &lsv1alpha1.DeployItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployitem",
					Namespace: state.Namespace,
				},
				Spec: lsv1alpha1.DeployItemSpec{
					Type: "some-type",
				},
			}

			err := state.Create(ctx, di)
			Expect(err).ToNot(HaveOccurred())

			updated := di.DeepCopy()
			updated.Spec.Type = "other-type"
			err = f.Client.Update(ctx, updated)
			Expect(err).To(HaveOccurred()) // validation webhook should have denied this
			Expect(err.Error()).To(HavePrefix("admission webhook \"deployitems.validation.landscaper.gardener.cloud\" denied the request"))
		})
	})
}
