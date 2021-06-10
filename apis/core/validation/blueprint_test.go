// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
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

		It("should fail if multiple ImportDefinition types are defined", func() {
			importDefinition := core.ImportDefinition{}
			importDefinition.Name = "myimport"
			importDefinition.TargetType = "test"
			importDefinition.Schema = &core.JSONSchemaDefinition{}

			allErrs := validation.ValidateBlueprintImportDefinitions(field.NewPath("x"), []core.ImportDefinition{importDefinition})
			Expect(allErrs).To(HaveLen(1))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("x[0][myimport]"),
			}))))
		})

		It("should fail if the config for the specified type is empty", func() {
			impDef1 := core.ImportDefinition{}
			impDef1.Name = "myimport1"
			impDef1.Type = core.ImportTypeData
			impDef2 := core.ImportDefinition{}
			impDef2.Name = "myimport2"
			impDef2.Type = core.ImportTypeTarget

			allErrs := validation.ValidateBlueprintImportDefinitions(field.NewPath("x"), []core.ImportDefinition{impDef1, impDef2})
			Expect(allErrs).To(HaveLen(2))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("x[0][myimport1]"),
				"Detail": ContainSubstring("schema"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("x[1][myimport2]"),
				"Detail": ContainSubstring("targetType"),
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

		It("should fail if subinstallation is defined by file and inline", func() {
			subinstallation := core.SubinstallationTemplate{
				File:                 "mypath",
				InstallationTemplate: &core.InstallationTemplate{},
			}

			allErrs := validation.ValidateSubinstallations(field.NewPath("b"), []core.SubinstallationTemplate{subinstallation})
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("b[0]"),
			}))))
		})

		It("should fail if a subinstallation is not defined by file or inline", func() {
			subinstallation := core.SubinstallationTemplate{}

			allErrs := validation.ValidateSubinstallations(field.NewPath("b"), []core.SubinstallationTemplate{subinstallation})
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("b[0]"),
			}))))
		})

		It("should fail if a secret or configmap reference is used in a InstallationTemplate", func() {
			tmpl := &core.InstallationTemplate{}
			tmpl.Imports.Data = []core.DataImport{
				{
					Name:      "myimport",
					SecretRef: &core.SecretReference{ObjectReference: core.ObjectReference{Name: "mysecret"}},
				},
				{
					Name:         "mysecondimport",
					ConfigMapRef: &core.ConfigMapReference{ObjectReference: core.ObjectReference{Name: "mycm"}},
				},
			}

			allErrs := validation.ValidateInstallationTemplate(field.NewPath("b"), tmpl)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("b.imports.data[0].secretRef"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("b.imports.data[1].configMapRef"),
			}))))
		})

		Context("Import Satisfaction", func() {
			It("should pass if a data import of a subinstallation is imported by its parent", func() {
				imports := []core.ImportDefinition{
					{
						FieldValueDefinition: core.FieldValueDefinition{
							Name:   "myimportref",
							Schema: &core.JSONSchemaDefinition{RawMessage: []byte("type: string")},
						},
					},
				}
				tmpl := &core.InstallationTemplate{}
				tmpl.Name = "my-inst"
				tmpl.Blueprint.Ref = "myref"
				tmpl.Imports.Data = []core.DataImport{
					{
						Name:    "myimport",
						DataRef: "myimportref",
					},
				}

				allErrs := validation.ValidateInstallationTemplates(field.NewPath("b"), imports, []*core.InstallationTemplate{tmpl})
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

				tmpl := &core.InstallationTemplate{}
				tmpl.Name = "my-inst"
				tmpl.Blueprint.Ref = "myref"
				tmpl.Imports.Targets = []core.TargetImportExport{
					{
						Name:   "myimport",
						Target: "myimportref",
					},
				}

				allErrs := validation.ValidateInstallationTemplates(field.NewPath("b"), imports, []*core.InstallationTemplate{tmpl})
				Expect(allErrs).To(HaveLen(0))
			})

			It("should fail if a data import of a subinstallation is not satisfied", func() {
				tmpl := &core.InstallationTemplate{}
				tmpl.Blueprint.Ref = "myref"
				tmpl.Imports.Data = []core.DataImport{
					{
						Name:    "myimport",
						DataRef: "myimportref",
					},
				}

				allErrs := validation.ValidateInstallationTemplates(field.NewPath("b"), nil, []*core.InstallationTemplate{tmpl})
				Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeNotFound),
					"Field": Equal("b[0].imports.data[0][myimport]"),
				}))))
			})

			It("should fail if a target import of a subinstallation is not satisfied", func() {
				tmpl := &core.InstallationTemplate{}
				tmpl.Blueprint.Ref = "myref"
				tmpl.Imports.Targets = []core.TargetImportExport{
					{
						Name:   "myimport",
						Target: "myimportref",
					},
				}

				allErrs := validation.ValidateInstallationTemplates(field.NewPath("b"), nil, []*core.InstallationTemplate{tmpl})
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
							Schema: &core.JSONSchemaDefinition{RawMessage: []byte("type: string")},
						},
					},
				}
				tmpl1 := &core.InstallationTemplate{}
				tmpl1.Blueprint.Ref = "myref"
				tmpl1.Exports.Data = []core.DataExport{
					{
						Name:    "myimport",
						DataRef: "myimportref",
					},
					{
						Name:    "mysecondexport",
						DataRef: "mysecondexportref",
					},
				}

				tmpl2 := &core.InstallationTemplate{}
				tmpl2.Blueprint.Ref = "myref"
				tmpl2.Exports.Data = []core.DataExport{
					{
						Name:    "mysecondexport",
						DataRef: "mysecondexportref",
					},
				}

				allErrs := validation.ValidateInstallationTemplates(
					field.NewPath("b"),
					imports,
					[]*core.InstallationTemplate{tmpl1, tmpl2})
				Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("b[0].exports.data[0][myimport]"),
				}))))
				Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("b[1].exports.data[0][mysecondexport]"),
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
				tmpl1 := &core.InstallationTemplate{}
				tmpl1.Blueprint.Ref = "myref"
				tmpl1.Exports.Targets = []core.TargetImportExport{
					{
						Name:   "myimport",
						Target: "myimportref",
					},
					{
						Name:   "mysecondexport",
						Target: "mysecondexportref",
					},
				}

				tmpl2 := &core.InstallationTemplate{}
				tmpl2.Blueprint.Ref = "myref"
				tmpl2.Exports.Targets = []core.TargetImportExport{
					{
						Name:   "mysecondexport",
						Target: "mysecondexportref",
					},
				}

				allErrs := validation.ValidateInstallationTemplates(
					field.NewPath("b"),
					imports,
					[]*core.InstallationTemplate{tmpl1, tmpl2})
				Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("b[0].exports.targets[0][myimport]"),
				}))))
				Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("b[1].exports.targets[0][mysecondexport]"),
				}))))
			})
		})
	})

})
