// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"os"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/core"
	"github.com/gardener/landscaper/apis/core/validation"
)

var _ = Describe("Blueprint", func() {

	Context("ImportDefinitions", func() {
		It("should pass if a ImportDefinition is valid", func() {
			importDefinition := core.ImportDefinition{}
			importDefinition.Name = "my-import"
			importDefinition.TargetType = "test"

			allErrs := validation.ValidateBlueprintImportDefinitions(field.NewPath(""), []core.ImportDefinition{importDefinition})
			Expect(allErrs).To(HaveLen(0))
		})

		It("should fail if ImportDefinition.name is empty", func() {
			importDefinition := core.ImportDefinition{}

			allErrs := validation.ValidateBlueprintImportDefinitions(field.NewPath("b"), []core.ImportDefinition{importDefinition})
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("b[0].name"),
			}))))
		})

		It("should fail if no ImportDefinition type is defined", func() {
			importDefinition := core.ImportDefinition{}
			importDefinition.Name = "myimport"

			allErrs := validation.ValidateBlueprintImportDefinitions(field.NewPath("b"), []core.ImportDefinition{importDefinition})
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("b[0][myimport]"),
			}))))
		})

		It("should fail if there are conditional imports on a required import", func() {
			importDefinition := core.ImportDefinition{}
			importDefinition.Name = "myimport"
			importDefinition.TargetType = "test"
			conImportDef := core.ImportDefinition{}
			conImportDef.Name = "myConditionalImport"
			conImportDef.TargetType = "test"
			importDefinition.ConditionalImports = []core.ImportDefinition{
				conImportDef,
			}

			allErrs := validation.ValidateBlueprintImportDefinitions(field.NewPath("x"), []core.ImportDefinition{importDefinition})
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("x[0][myimport]"),
				"Detail": Equal("conditional imports on required import"),
			}))))
		})
	})

	Context("ExportDefinitions", func() {
		It("should pass if a ExportDefinitions is valid", func() {
			exportDefinition := core.ExportDefinition{}
			exportDefinition.Name = "my-import"
			exportDefinition.TargetType = "test"

			allErrs := validation.ValidateBlueprintExportDefinitions(field.NewPath(""), []core.ExportDefinition{exportDefinition})
			Expect(allErrs).To(HaveLen(0))
		})

		It("should fail if ExportDefinitions.name is empty", func() {
			exportDefinition := core.ExportDefinition{}

			allErrs := validation.ValidateBlueprintExportDefinitions(field.NewPath("b"), []core.ExportDefinition{exportDefinition})
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("b[0].name"),
			}))))
		})

		It("should fail if no ExportDefinitions type is defined", func() {
			exportDefinition := core.ExportDefinition{}
			exportDefinition.Name = "myimport"

			allErrs := validation.ValidateBlueprintExportDefinitions(field.NewPath("b"), []core.ExportDefinition{exportDefinition})
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("b[0][myimport]"),
			}))))
		})
	})

	Context("TemplateExecutor", func() {
		It("should pass if a TemplateExecutor is valid", func() {
			executor := core.TemplateExecutor{}
			executor.Name = "myname"
			executor.Type = "mytype"

			allErrs := validation.ValidateTemplateExecutorList(field.NewPath(""), []core.TemplateExecutor{executor})
			Expect(allErrs).To(HaveLen(0))
		})

		It("should fail if TemplateExecutor.name is missing", func() {
			executor := core.TemplateExecutor{}

			allErrs := validation.ValidateTemplateExecutorList(field.NewPath("b"), []core.TemplateExecutor{executor})
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("b[0].name"),
			}))))
		})

		It("should fail if TemplateExecutor.type is missing", func() {
			executor := core.TemplateExecutor{}
			executor.Name = "myname"

			allErrs := validation.ValidateTemplateExecutorList(field.NewPath("b"), []core.TemplateExecutor{executor})
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("b[0][myname].type"),
			}))))
		})
	})

	Context("InstallationTemplate", func() {
		It("should pass if a InstallationTemplate is valid", func() {
			installationTemplate := &core.InstallationTemplate{}
			installationTemplate.Name = "myname"
			installationTemplate.Blueprint = core.InstallationTemplateBlueprintDefinition{
				Ref: "my-ref",
			}

			allErrs := validation.ValidateInstallationTemplate(field.NewPath(""), installationTemplate)
			Expect(allErrs).To(HaveLen(0))
		})

		It("should fail if InstallationTemplate.name is missing", func() {
			installationTemplate := &core.InstallationTemplate{}

			allErrs := validation.ValidateInstallationTemplate(field.NewPath("b"), installationTemplate)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("b.name"),
			}))))
		})

		It("should fail if InstallationTemplate.name is invalid", func() {
			installationTemplate := &core.InstallationTemplate{}
			installationTemplate.Name = "%$.-"

			allErrs := validation.ValidateInstallationTemplate(field.NewPath("b"), installationTemplate)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("b.name"),
			}))))
		})

		It("should fail if InstallationTemplate.blueprint is missing", func() {
			installationTemplate := &core.InstallationTemplate{}
			installationTemplate.Name = "myname"

			allErrs := validation.ValidateInstallationTemplate(field.NewPath("b"), installationTemplate)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("b.blueprint"),
			}))))
		})
	})

	Context("Subinstallations", func() {
		It("should pass if a InstallationTemplate defined by a file is valid", func() {
			subinstallation := core.SubinstallationTemplate{
				File: "mypath",
			}
			fs := memoryfs.New()
			installationTemplateBytes := []byte(`
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: InstallationTemplate
name: my-tmpl
blueprint:
  ref: myref
`)
			Expect(vfs.WriteFile(fs, "mypath", installationTemplateBytes, os.ModePerm)).To(Succeed())

			allErrs := validation.ValidateSubinstallations(field.NewPath("b"), fs, []core.ImportDefinition{}, []core.SubinstallationTemplate{subinstallation})
			Expect(allErrs).To(HaveLen(0))
		})

		It("should fail if subinstallation is defined by file and inline", func() {
			subinstallation := core.SubinstallationTemplate{
				File:                 "mypath",
				InstallationTemplate: &core.InstallationTemplate{},
			}

			allErrs := validation.ValidateSubinstallations(field.NewPath("b"), memoryfs.New(), []core.ImportDefinition{}, []core.SubinstallationTemplate{subinstallation})
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("b[0]"),
			}))))
		})

		It("should fail if a subinstallation is not defined by file or inline", func() {
			subinstallation := core.SubinstallationTemplate{}

			allErrs := validation.ValidateSubinstallations(field.NewPath("b"), memoryfs.New(), []core.ImportDefinition{}, []core.SubinstallationTemplate{subinstallation})
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("b[0]"),
			}))))
		})

		It("should fail if the defined file path does not exist ", func() {
			subinstallation := core.SubinstallationTemplate{
				File: "mypath",
			}

			allErrs := validation.ValidateSubinstallations(field.NewPath("b"), memoryfs.New(), []core.ImportDefinition{}, []core.SubinstallationTemplate{subinstallation})
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotFound),
				"Field": Equal("b[0].file"),
			}))))
		})

		It("should fail if a InstallationTemplate defined by a file is invalid", func() {
			subinstallation := core.SubinstallationTemplate{
				File:                 "mypath",
				InstallationTemplate: &core.InstallationTemplate{},
			}
			fs := memoryfs.New()
			installationTemplateBytes := []byte(`wrong type`)
			Expect(vfs.WriteFile(fs, "mypath", installationTemplateBytes, os.ModePerm)).To(Succeed())

			allErrs := validation.ValidateSubinstallations(field.NewPath("b"), fs, []core.ImportDefinition{}, []core.SubinstallationTemplate{subinstallation})
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("b[0]"),
			}))))
		})

		It("should fail if a secret or configmap reference is used in a InstallationTemplate", func() {
			subinstallation := core.SubinstallationTemplate{
				File: "mypath",
			}
			fs := memoryfs.New()
			installationTemplateBytes := []byte(`
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: InstallationTemplate
name: my-tmpl
blueprint:
  ref: myref
imports:
  data:
  - name: myimport
    secretRef:
      name: mysecret
  - name: mysecondimport
    configMapRef:
      name: mycm
`)
			Expect(vfs.WriteFile(fs, "mypath", installationTemplateBytes, os.ModePerm)).To(Succeed())

			allErrs := validation.ValidateSubinstallations(field.NewPath("b"), fs, []core.ImportDefinition{}, []core.SubinstallationTemplate{subinstallation})
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("b[0].imports.data[0].secretRef"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("b[0].imports.data[1].configMapRef"),
			}))))
		})

		Context("Import Satisfaction", func() {
			It("should pass if a data import of a subinstallation is imported by its parent", func() {
				imports := []core.ImportDefinition{
					{
						FieldValueDefinition: core.FieldValueDefinition{
							Name:   "myimportref",
							Schema: []byte("type: string"),
						},
					},
				}
				subinstallation := core.SubinstallationTemplate{
					File: "mypath",
				}
				fs := memoryfs.New()
				installationTemplateBytes := []byte(`
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: InstallationTemplate
name: my-tmpl
blueprint:
  ref: myref
imports:
  data:
  - name: myimport
    dataRef: myimportref
`)
				Expect(vfs.WriteFile(fs, "mypath", installationTemplateBytes, os.ModePerm)).To(Succeed())

				allErrs := validation.ValidateSubinstallations(field.NewPath("b"), fs, imports, []core.SubinstallationTemplate{subinstallation})
				Expect(allErrs).To(HaveLen(0))
			})

			It("should pass if a target import of a subinstallation is imported by its parent", func() {
				imports := []core.ImportDefinition{
					{
						FieldValueDefinition: core.FieldValueDefinition{
							Name:       "myimportref",
							TargetType: "mytype",
						},
					},
				}
				subinstallation := core.SubinstallationTemplate{
					File: "mypath",
				}
				fs := memoryfs.New()
				installationTemplateBytes := []byte(`
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: InstallationTemplate
name: my-tmpl
blueprint:
  ref: myref
imports:
  targets:
  - name: myimport
    target: myimportref
`)
				Expect(vfs.WriteFile(fs, "mypath", installationTemplateBytes, os.ModePerm)).To(Succeed())

				allErrs := validation.ValidateSubinstallations(field.NewPath("b"), fs, imports, []core.SubinstallationTemplate{subinstallation})
				Expect(allErrs).To(HaveLen(0))
			})

			It("should fail if a data import of a subinstallation is not satisfied", func() {
				subinstallation := core.SubinstallationTemplate{
					File: "mypath",
				}
				fs := memoryfs.New()
				installationTemplateBytes := []byte(`
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: InstallationTemplate
name: my-tmpl
blueprint:
  ref: myref
imports:
  data:
  - name: myimport
    dataRef: myimportref
`)
				Expect(vfs.WriteFile(fs, "mypath", installationTemplateBytes, os.ModePerm)).To(Succeed())

				allErrs := validation.ValidateSubinstallations(field.NewPath("b"), fs, []core.ImportDefinition{}, []core.SubinstallationTemplate{subinstallation})
				Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeNotFound),
					"Field": Equal("b[0].imports.data[0][myimport]"),
				}))))
			})

			It("should fail if a target import of a subinstallation is not satisfied", func() {
				subinstallation := core.SubinstallationTemplate{
					File: "mypath",
				}
				fs := memoryfs.New()
				installationTemplateBytes := []byte(`
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: InstallationTemplate
name: my-tmpl
blueprint:
  ref: myref
imports:
  targets:
  - name: myimport
    target: myimportref
`)
				Expect(vfs.WriteFile(fs, "mypath", installationTemplateBytes, os.ModePerm)).To(Succeed())

				allErrs := validation.ValidateSubinstallations(field.NewPath("b"), fs, []core.ImportDefinition{}, []core.SubinstallationTemplate{subinstallation})
				Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeNotFound),
					"Field": Equal("b[0].imports.targets[0][myimport]"),
				}))))
			})

			It("should fail if a subinstallation exports a already defined data object", func() {
				imports := []core.ImportDefinition{
					{
						FieldValueDefinition: core.FieldValueDefinition{
							Name:   "myimportref",
							Schema: []byte("type: string"),
						},
					},
				}
				subinstallations := []core.SubinstallationTemplate{
					{
						File: "mypath",
					},
					{
						File: "mypath2",
					},
				}
				fs := memoryfs.New()
				installationTemplateBytes := []byte(`
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: InstallationTemplate
name: my-tmpl
blueprint:
  ref: myref
exports:
  data:
  - name: myImport
    dataRef: myimportref
  - name: mySecondImport
    dataRef: mysecondimportref
`)
				Expect(vfs.WriteFile(fs, "mypath", installationTemplateBytes, os.ModePerm)).To(Succeed())
				installationTemplateBytes = []byte(`
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: InstallationTemplate
name: my-tmpl2
blueprint:
  ref: myref
exports:
  data:
  - name: mySecondImport
    dataRef: mysecondimportref
`)
				Expect(vfs.WriteFile(fs, "mypath2", installationTemplateBytes, os.ModePerm)).To(Succeed())

				allErrs := validation.ValidateSubinstallations(field.NewPath("b"), fs, imports, subinstallations)
				Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("b[0].exports.data[0][myImport]"),
				}))))
				Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("b[1].exports.data[0][mySecondImport]"),
				}))))
			})

			It("should fail if a subinstallation exports a already defined target", func() {
				imports := []core.ImportDefinition{
					{
						FieldValueDefinition: core.FieldValueDefinition{
							Name:       "myimportref",
							TargetType: "mytype",
						},
					},
				}
				subinstallations := []core.SubinstallationTemplate{
					{
						File: "mypath",
					},
					{
						File: "mypath2",
					},
				}
				fs := memoryfs.New()
				installationTemplateBytes := []byte(`
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: InstallationTemplate
name: my-tmpl
blueprint:
  ref: myref
exports:
  targets:
  - name: myImport
    target: myimportref
  - name: mySecondImport
    target: mysecondimportref
`)
				Expect(vfs.WriteFile(fs, "mypath", installationTemplateBytes, os.ModePerm)).To(Succeed())
				installationTemplateBytes = []byte(`
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: InstallationTemplate
name: my-tmpl2
blueprint:
  ref: myref
exports:
  targets:
  - name: mySecondImport
    target: mysecondimportref
`)
				Expect(vfs.WriteFile(fs, "mypath2", installationTemplateBytes, os.ModePerm)).To(Succeed())

				allErrs := validation.ValidateSubinstallations(field.NewPath("b"), fs, imports, subinstallations)
				Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("b[0].exports.targets[0][myImport]"),
				}))))
				Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("b[1].exports.targets[0][mySecondImport]"),
				}))))
			})
		})
	})

})
