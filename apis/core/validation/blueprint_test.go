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
			impDef1 := core.ImportDefinition{}
			impDef1.Name = "my-import1"
			impDef1.Type = core.ImportTypeTarget
			impDef1.TargetType = "test"
			impDef2 := core.ImportDefinition{}
			impDef2.Name = "my-import2"
			impDef2.Type = core.ImportTypeTargetList
			impDef2.TargetType = "test"
			impDef3 := core.ImportDefinition{}
			impDef3.Name = "my-import3"
			impDef3.Type = core.ImportTypeData
			impDef3.Schema = &core.JSONSchemaDefinition{}

			allErrs := validation.ValidateBlueprintImportDefinitions(field.NewPath(""), []core.ImportDefinition{impDef1, impDef2, impDef3})
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

		It("should fail if multiple ImportDefinition types are defined (legacy format)", func() {
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
			impDef3 := core.ImportDefinition{}
			impDef3.Name = "myimport3"
			impDef3.Type = core.ImportTypeTargetList

			allErrs := validation.ValidateBlueprintImportDefinitions(field.NewPath("x"), []core.ImportDefinition{impDef1, impDef2, impDef3})
			Expect(allErrs).To(HaveLen(3))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("x[0][myimport1]"),
				"Detail": ContainSubstring("Schema"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("x[1][myimport2]"),
				"Detail": ContainSubstring("TargetType"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("x[2][myimport3]"),
				"Detail": ContainSubstring("TargetType"),
			}))))
		})

		It("should fail a wrong config for the specified type is given", func() {
			impDefs := []core.ImportDefinition{
				{
					FieldValueDefinition: core.FieldValueDefinition{
						Name:       "myimport1",
						Schema:     &core.JSONSchemaDefinition{},
						TargetType: "test",
					},
					Type: core.ImportTypeData,
				},
				{
					FieldValueDefinition: core.FieldValueDefinition{
						Name:       "myimport2",
						Schema:     &core.JSONSchemaDefinition{},
						TargetType: "test",
					},
					Type: core.ImportTypeTarget,
				},
				{
					FieldValueDefinition: core.FieldValueDefinition{
						Name:       "myimport3",
						Schema:     &core.JSONSchemaDefinition{},
						TargetType: "test",
					},
					Type: core.ImportTypeTargetList,
				},
			}

			allErrs := validation.ValidateBlueprintImportDefinitions(field.NewPath("x"), impDefs)
			Expect(allErrs).To(HaveLen(3))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("x[0][myimport1]"),
				"Detail": ContainSubstring("TargetType"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("x[1][myimport2]"),
				"Detail": ContainSubstring("Schema"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("x[2][myimport3]"),
				"Detail": ContainSubstring("Schema"),
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
			expDef1 := core.ExportDefinition{}
			expDef1.Name = "my-export1"
			expDef1.Type = core.ExportTypeTarget
			expDef1.TargetType = "test"
			expDef2 := core.ExportDefinition{}
			expDef2.Name = "my-export3"
			expDef2.Type = core.ExportTypeData
			expDef2.Schema = &core.JSONSchemaDefinition{}

			allErrs := validation.ValidateBlueprintExportDefinitions(field.NewPath(""), []core.ExportDefinition{expDef1, expDef2})
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
					SecretRef: &core.LocalSecretReference{Name: "mysecret"},
				},
				{
					Name:         "mysecondimport",
					ConfigMapRef: &core.LocalConfigMapReference{Name: "mycm"},
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
				tmpl.Imports.Targets = []core.TargetImport{
					{
						Name:   "myimport",
						Target: "myimportref",
					},
				}

				allErrs := validation.ValidateInstallationTemplates(field.NewPath("b"), imports, []*core.InstallationTemplate{tmpl})
				Expect(allErrs).To(HaveLen(0))
			})

			It("should pass if a targetlist import of a subinstallation refers to a targetlist imported by its parent", func() {
				imports := []core.ImportDefinition{
					{
						FieldValueDefinition: core.FieldValueDefinition{
							Name:       "myimportref",
							TargetType: "mytype",
						},
						Type: core.ImportTypeTargetList,
					},
				}

				tmpl := &core.InstallationTemplate{}
				tmpl.Name = "my-inst"
				tmpl.Blueprint.Ref = "myref"
				tmpl.Imports.Targets = []core.TargetImport{
					{
						Name:                "myimport",
						TargetListReference: "myimportref",
					},
				}

				allErrs := validation.ValidateInstallationTemplates(field.NewPath("b"), imports, []*core.InstallationTemplate{tmpl})
				Expect(allErrs).To(HaveLen(0))
			})

			It("should pass if a targetmap import of a subinstallation refers to a targetmap imported by its parent", func() {
				imports := []core.ImportDefinition{
					{
						FieldValueDefinition: core.FieldValueDefinition{
							Name:       "myimportref",
							TargetType: "mytype",
						},
						Type: core.ImportTypeTargetMap,
					},
				}

				tmpl := &core.InstallationTemplate{}
				tmpl.Name = "my-inst"
				tmpl.Blueprint.Ref = "myref"
				tmpl.Imports.Targets = []core.TargetImport{
					{
						Name:               "myimport",
						TargetMapReference: "myimportref",
					},
				}

				allErrs := validation.ValidateInstallationTemplates(field.NewPath("b"), imports, []*core.InstallationTemplate{tmpl})
				Expect(allErrs).To(HaveLen(0))
			})

			It("should pass if a targetmap import of a subinstallation refers to targets imported by its parent", func() {
				imports := []core.ImportDefinition{
					{
						FieldValueDefinition: core.FieldValueDefinition{
							Name:       "myimportref",
							TargetType: "mytype",
						},
						Type: core.ImportTypeTarget,
					},
				}

				tmpl := &core.InstallationTemplate{}
				tmpl.Name = "my-inst"
				tmpl.Blueprint.Ref = "myref"
				tmpl.Imports.Targets = []core.TargetImport{
					{
						Name: "myimport",
						TargetMap: map[string]string{
							"mykey1": "myimportref",
						},
					},
				}

				allErrs := validation.ValidateInstallationTemplates(field.NewPath("b"), imports, []*core.InstallationTemplate{tmpl})
				Expect(allErrs).To(HaveLen(0))
			})

			It("should pass if a targetmap import of a subinstallation refers to targets exported by a sibling", func() {
				imports := []core.ImportDefinition{
					{
						FieldValueDefinition: core.FieldValueDefinition{
							Name:   "myimportref",
							Schema: &core.JSONSchemaDefinition{RawMessage: []byte("type: string")},
						},
					},
				}
				// subinstallation template exporting a target
				tmpl1 := &core.InstallationTemplate{}
				tmpl1.Name = "my-inst1"
				tmpl1.Blueprint.Ref = "myref1"
				tmpl1.Exports.Targets = []core.TargetExport{
					{
						Name:   "myexport",
						Target: "myexportref",
					},
				}
				// subinstallation template importing a targetmap
				tmpl2 := &core.InstallationTemplate{}
				tmpl2.Name = "my-inst2"
				tmpl2.Blueprint.Ref = "myref2"
				tmpl2.Imports.Targets = []core.TargetImport{
					{
						Name: "myimport",
						TargetMap: map[string]string{
							"mykey": "myexportref",
						},
					},
				}

				allErrs := validation.ValidateInstallationTemplates(field.NewPath("b"), imports, []*core.InstallationTemplate{tmpl1, tmpl2})
				Expect(allErrs).To(HaveLen(0))
			})

			It("should pass if a target import of a subinstallation is imported as part of a targetlist by its parent", func() {
				imports := []core.ImportDefinition{
					{
						FieldValueDefinition: core.FieldValueDefinition{
							Name:       "myimportref",
							TargetType: "mytype",
						},
						Type: core.ImportTypeTargetList,
					},
				}

				tmpl := &core.InstallationTemplate{}
				tmpl.Name = "my-inst"
				tmpl.Blueprint.Ref = "myref"
				tmpl.Imports.Targets = []core.TargetImport{
					{
						Name:   "myimport",
						Target: "myimportref[1]",
					},
				}

				allErrs := validation.ValidateInstallationTemplates(field.NewPath("b"), imports, []*core.InstallationTemplate{tmpl})
				Expect(allErrs).To(HaveLen(0))
			})

			It("should pass if a target import of a subinstallation is imported as part of a targetmap by its parent", func() {
				imports := []core.ImportDefinition{
					{
						FieldValueDefinition: core.FieldValueDefinition{
							Name:       "myimportref",
							TargetType: "mytype",
						},
						Type: core.ImportTypeTargetMap,
					},
				}

				tmpl := &core.InstallationTemplate{}
				tmpl.Name = "my-inst"
				tmpl.Blueprint.Ref = "myref"
				tmpl.Imports.Targets = []core.TargetImport{
					{
						Name:   "myimport",
						Target: "myimportref[mykey]",
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
				tmpl.Imports.Targets = []core.TargetImport{
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

			It("should fail if a target import of a subinstallation refers to a complete targetlist from its parent", func() {
				imports := []core.ImportDefinition{
					{
						FieldValueDefinition: core.FieldValueDefinition{
							Name:       "myimportref",
							TargetType: "mytype",
						},
						Type: core.ImportTypeTargetList,
					},
				}

				tmpl := &core.InstallationTemplate{}
				tmpl.Blueprint.Ref = "myref"
				tmpl.Imports.Targets = []core.TargetImport{
					{
						Name:   "myimport",
						Target: "myimportref",
					},
				}

				allErrs := validation.ValidateInstallationTemplates(field.NewPath("b"), imports, []*core.InstallationTemplate{tmpl})
				Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeNotFound),
					"Field": Equal("b[0].imports.targets[0][myimport]"),
				}))))
			})

			It("should fail if a target import of a subinstallation refers to a complete targetmap from its parent", func() {
				imports := []core.ImportDefinition{
					{
						FieldValueDefinition: core.FieldValueDefinition{
							Name:       "myimportref",
							TargetType: "mytype",
						},
						Type: core.ImportTypeTargetMap,
					},
				}

				tmpl := &core.InstallationTemplate{}
				tmpl.Blueprint.Ref = "myref"
				tmpl.Imports.Targets = []core.TargetImport{
					{
						Name:   "myimport",
						Target: "myimportref",
					},
				}

				allErrs := validation.ValidateInstallationTemplates(field.NewPath("b"), imports, []*core.InstallationTemplate{tmpl})
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
					"Field": Equal("b[0].exports.data[0][myimport/myimportref]"),
				}))))
				Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("b[1].exports.data[0][mysecondexport/mysecondexportref]"),
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
				tmpl1.Exports.Targets = []core.TargetExport{
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
				tmpl2.Exports.Targets = []core.TargetExport{
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
					"Field": Equal("b[0].exports.targets[0][myimport/myimportref]"),
				}))))
				Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("b[1].exports.targets[0][mysecondexport/mysecondexportref]"),
				}))))
			})
		})
	})

})
