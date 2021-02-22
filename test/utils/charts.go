// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/json"
	"io/ioutil"
	"os"

	. "github.com/onsi/gomega"

	chartloader "helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// ReadChartFrom reads a Helm chart from a filesystem path and returns it as raw bytes array
func ReadChartFrom(path string) ([]byte, func()) {
	chart, err := chartloader.LoadDir(path)
	Expect(err).ToNot(HaveOccurred())
	tempDir, err := ioutil.TempDir(os.TempDir(), "chart-")
	Expect(err).ToNot(HaveOccurred())
	closer := func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	}

	chartPath, err := chartutil.Save(chart, tempDir)
	Expect(err).ToNot(HaveOccurred())

	chartBytes, err := ioutil.ReadFile(chartPath)
	Expect(err).ToNot(HaveOccurred())
	return chartBytes, closer
}

// ReadValuesFromFile reads Helm values from a file and returns them as JSON raw message
func ReadValuesFromFile(path string) json.RawMessage {
	var values json.RawMessage
	in, err := ioutil.ReadFile(path)
	Expect(err).ToNot(HaveOccurred())
	values, err = yaml.YAMLToJSON(in)
	Expect(err).ToNot(HaveOccurred())
	return values
}

// InjectTargetSelectorIntoValues injects (and potentially overwrites) a TargetSelector into a values JSON raw message
func InjectTargetSelectorIntoValues(values *json.RawMessage, targetSelector []lsv1alpha1.TargetSelector) {
	v := make(map[string]interface{})
	err := json.Unmarshal(*values, &v)
	Expect(err).ToNot(HaveOccurred())
	v["targetSelector"] = targetSelector
	*values, err = json.Marshal(v)
	Expect(err).ToNot(HaveOccurred())
}

// InjectImageTagIntoValues injects (and potentially overwrites) an Image tag into a values JSON raw message
func InjectImageTagIntoValues(values *json.RawMessage, imageTag string) {
	v := make(map[string]interface{})
	err := json.Unmarshal(*values, &v)
	Expect(err).ToNot(HaveOccurred())
	v["image"].(map[string]interface{})["tag"] = imageTag
	*values, err = json.Marshal(v)
	Expect(err).ToNot(HaveOccurred())
}
