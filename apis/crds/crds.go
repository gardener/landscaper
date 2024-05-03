// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package crds

import (
	"embed"
	"fmt"
	"path/filepath"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

//go:embed manifests/landscaper.gardener.cloud*.yaml
var CRDFS embed.FS

// CRDs returns the generated CustomResourceDefinitions as go structs.
// Panics if anything goes wrong trying to read the CRD files, as an error here is likely related to a wrong build or invalid generated CRDs.
func CRDs() []*apiextv1.CustomResourceDefinition {
	files, err := CRDFS.ReadDir("manifests")
	if err != nil {
		panic(err)
	}
	res := []*apiextv1.CustomResourceDefinition{}
	for _, f := range files {
		data, err := CRDFS.ReadFile(filepath.Join("manifests", f.Name()))
		if err != nil {
			panic(fmt.Errorf("error reading CRD file '%s': %w", f.Name(), err))
		}
		crd := &apiextv1.CustomResourceDefinition{}
		if err := yaml.Unmarshal(data, crd); err != nil {
			panic(fmt.Errorf("error parsing file '%s' into CRD: %w", f.Name(), err))
		}
		res = append(res, crd)
	}
	return res
}
