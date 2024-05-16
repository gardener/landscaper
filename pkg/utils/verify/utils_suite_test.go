// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package verify_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/utils/verify"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Utils Test Suite")
}

var _ = Describe("Verify Utils", func() {

	Describe("Correctly detect the enablement state of verification", func() {

		Context("config enforces verify", func() {
			It("should return true although installation does not specify verification info", func() {
				config := &config.LandscaperConfiguration{
					SignatureVerificationEnforcementPolicy: config.Enforce,
				}
				inst := &lsv1alpha1.Installation{
					Spec: lsv1alpha1.InstallationSpec{
						Verification: nil,
					},
				}
				Expect(verify.IsVerifyEnabled(inst, config)).To(BeTrue())
			})
			It("should return true even if installation specify verification info", func() {
				config := &config.LandscaperConfiguration{
					SignatureVerificationEnforcementPolicy: config.Enforce,
				}
				inst := &lsv1alpha1.Installation{
					Spec: lsv1alpha1.InstallationSpec{
						Verification: &lsv1alpha1.Verification{
							SignatureName: "test-signature-name",
						},
					},
				}
				Expect(verify.IsVerifyEnabled(inst, config)).To(BeTrue())
			})
		})
		Context("config explicitly disables verify", func() {
			It("should return false even if installation does not specify verification info", func() {
				config := &config.LandscaperConfiguration{
					SignatureVerificationEnforcementPolicy: config.Disabled,
				}
				inst := &lsv1alpha1.Installation{
					Spec: lsv1alpha1.InstallationSpec{
						Verification: nil,
					},
				}
				Expect(verify.IsVerifyEnabled(inst, config)).To(BeFalse())
			})
			It("should return false although installation specify verification info", func() {
				config := &config.LandscaperConfiguration{
					SignatureVerificationEnforcementPolicy: config.Disabled,
				}
				inst := &lsv1alpha1.Installation{
					Spec: lsv1alpha1.InstallationSpec{
						Verification: &lsv1alpha1.Verification{
							SignatureName: "test-signature-name",
						},
					},
				}
				Expect(verify.IsVerifyEnabled(inst, config)).To(BeFalse())
			})
		})
		Context("config does not enforce verification, it depends on the installation", func() {
			It("should return false if installation does not specify verification info", func() {
				config := &config.LandscaperConfiguration{
					SignatureVerificationEnforcementPolicy: config.DoNotEnforce,
				}
				inst := &lsv1alpha1.Installation{
					Spec: lsv1alpha1.InstallationSpec{
						Verification: nil,
					},
				}
				Expect(verify.IsVerifyEnabled(inst, config)).To(BeFalse())
			})
			It("should return true if installation specify verification info", func() {
				config := &config.LandscaperConfiguration{
					SignatureVerificationEnforcementPolicy: config.DoNotEnforce,
				}
				inst := &lsv1alpha1.Installation{
					Spec: lsv1alpha1.InstallationSpec{
						Verification: &lsv1alpha1.Verification{
							SignatureName: "test-signature-name",
						},
					},
				}
				Expect(verify.IsVerifyEnabled(inst, config)).To(BeTrue())
			})
		})

	})

	Describe("extract verification information", func() {
		var (
			ctx    context.Context
			state  *envtest.State
			client client.Client
		)
		signatureName := "signature-test-name"
		BeforeEach(func() {
			ctx = context.Background()
			var err error
			client, state, err = envtest.NewFakeClientFromPath("")
			utils.ExpectNoError(err)
			ns := &corev1.Namespace{}
			ns.GenerateName = "tests-"
			utils.ExpectNoError(state.Create(ctx, ns))
			state.Namespace = ns.Name
		})
		Context("missing verification information in installation", func() {
			It("should fail", func() {
				inst := &lsv1alpha1.Installation{
					Spec: lsv1alpha1.InstallationSpec{},
				}
				lscontext := &lsv1alpha1.Context{
					ContextConfiguration: lsv1alpha1.ContextConfiguration{},
				}

				_, _, _, err := verify.ExtractVerifyInfo(ctx, inst, lscontext, client)
				Expect(err).ToNot(BeNil())
			})
		})
		Context("instalaltion verification sgiantureName is empty", func() {
			It("should fail", func() {
				inst := &lsv1alpha1.Installation{
					Spec: lsv1alpha1.InstallationSpec{
						Verification: &lsv1alpha1.Verification{
							SignatureName: "",
						},
					},
				}
				lscontext := &lsv1alpha1.Context{
					ContextConfiguration: lsv1alpha1.ContextConfiguration{},
				}

				_, _, _, err := verify.ExtractVerifyInfo(ctx, inst, lscontext, client)
				Expect(err).ToNot(BeNil())
			})
		})
		Context("inst contains signature name but context missing entry for it", func() {
			It("should succeed", func() {
				inst := &lsv1alpha1.Installation{
					Spec: lsv1alpha1.InstallationSpec{
						Verification: &lsv1alpha1.Verification{
							SignatureName: signatureName,
						},
					},
				}
				lscontext := &lsv1alpha1.Context{
					ContextConfiguration: lsv1alpha1.ContextConfiguration{},
				}

				_, _, _, err := verify.ExtractVerifyInfo(ctx, inst, lscontext, client)
				Expect(err).ToNot(BeNil())
			})
		})
		Context("contains public key data", func() {
			It("should succeed with signature and public key data returned", func() {
				secretName := "publicKeySecret"
				inst := &lsv1alpha1.Installation{
					Spec: lsv1alpha1.InstallationSpec{
						Verification: &lsv1alpha1.Verification{
							SignatureName: signatureName,
						},
					},
				}
				lscontext := &lsv1alpha1.Context{
					ContextConfiguration: lsv1alpha1.ContextConfiguration{
						VerificationSignatures: map[string]lsv1alpha1.VerificationSignature{
							signatureName: lsv1alpha1.VerificationSignature{
								PublicKeySecretReference: &lsv1alpha1.SecretReference{
									ObjectReference: lsv1alpha1.ObjectReference{
										Name:      secretName,
										Namespace: state.Namespace,
									},
									Key: "key",
								},
							},
						},
					},
				}
				secret := &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      secretName,
						Namespace: state.Namespace,
					},
					Data: map[string][]byte{
						"key": []byte("test"),
					},
				}
				Expect(state.Create(ctx, secret)).To(Succeed())

				signame, publicKeyData, caCertData, err := verify.ExtractVerifyInfo(ctx, inst, lscontext, client)
				Expect(err).To(BeNil())
				Expect(signame).To(Equal(signatureName))
				Expect(publicKeyData).ToNot(BeNil())
				Expect(caCertData).To(BeNil())
			})
		})
		Context("contains ca cert data", func() {
			It("should succeed with signature and ca cert data returned", func() {
				secretName := "caCertSecret"
				inst := &lsv1alpha1.Installation{
					Spec: lsv1alpha1.InstallationSpec{
						Verification: &lsv1alpha1.Verification{
							SignatureName: signatureName,
						},
					},
				}
				lscontext := &lsv1alpha1.Context{
					ContextConfiguration: lsv1alpha1.ContextConfiguration{
						VerificationSignatures: map[string]lsv1alpha1.VerificationSignature{
							signatureName: {
								CaCertificateSecretReference: &lsv1alpha1.SecretReference{
									ObjectReference: lsv1alpha1.ObjectReference{
										Name:      secretName,
										Namespace: state.Namespace,
									},
									Key: "key",
								},
							},
						},
					},
				}

				secret := &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      secretName,
						Namespace: state.Namespace,
					},
					Data: map[string][]byte{
						"key": []byte("test"),
					},
				}
				Expect(state.Create(ctx, secret)).To(Succeed())

				signame, publicKeyData, caCertData, err := verify.ExtractVerifyInfo(ctx, inst, lscontext, client)
				Expect(err).To(BeNil())
				Expect(signame).To(Equal(signatureName))
				Expect(publicKeyData).To(BeNil())
				Expect(caCertData).ToNot(BeNil())
			})
		})
		Context("contains public key and ca cert data", func() {
			It("should succeed with signature, public key and ca cert data returned", func() {
				secretNamePublicKeyRef := "publicKeyRefSecret"
				secretNameCaCertRef := "caCertRefSecret"
				inst := &lsv1alpha1.Installation{
					Spec: lsv1alpha1.InstallationSpec{
						Verification: &lsv1alpha1.Verification{
							SignatureName: signatureName,
						},
					},
				}
				lscontext := &lsv1alpha1.Context{
					ContextConfiguration: lsv1alpha1.ContextConfiguration{
						VerificationSignatures: map[string]lsv1alpha1.VerificationSignature{
							signatureName: {
								PublicKeySecretReference: &lsv1alpha1.SecretReference{
									ObjectReference: lsv1alpha1.ObjectReference{
										Name:      secretNamePublicKeyRef,
										Namespace: state.Namespace,
									},
									Key: "key",
								},
								CaCertificateSecretReference: &lsv1alpha1.SecretReference{
									ObjectReference: lsv1alpha1.ObjectReference{
										Name:      secretNameCaCertRef,
										Namespace: state.Namespace,
									},
									Key: "key",
								},
							},
						},
					},
				}

				secretPublicKey := &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      secretNamePublicKeyRef,
						Namespace: state.Namespace,
					},
					Data: map[string][]byte{
						"key": []byte("test"),
					},
				}
				Expect(state.Create(ctx, secretPublicKey)).To(Succeed())
				secretCaCert := &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      secretNameCaCertRef,
						Namespace: state.Namespace,
					},
					Data: map[string][]byte{
						"key": []byte("test"),
					},
				}
				Expect(state.Create(ctx, secretCaCert)).To(Succeed())

				signame, publicKeyData, caCertData, err := verify.ExtractVerifyInfo(ctx, inst, lscontext, client)
				Expect(err).To(BeNil())
				Expect(signame).To(Equal(signatureName))
				Expect(publicKeyData).ToNot(BeNil())
				Expect(caCertData).ToNot(BeNil())
			})
		})

	})
})
