// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helper

import (
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

const Base32EncodeStdLowerCase = "abcdefghijklmnopqrstuvwxyz234567"

const SourceDelimiter = "/"

// GenerateDataObjectName generates the unique name for a data object exported or imported by a installation.
func GenerateDataObjectName(context string, key string) string {
	name := fmt.Sprintf("%s/%s", context, key)
	h := sha1.New()
	_, _ = h.Write([]byte(name))
	// we need base32 encoding as some base64 (even url safe base64) characters are not supported by k8s
	// see https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
	return base32.NewEncoding(Base32EncodeStdLowerCase).WithPadding(base32.NoPadding).EncodeToString(h.Sum(nil))
}

// DataObjectSourceFromObject returns the data object source for a runtime object.
func DataObjectSourceFromObject(src runtime.Object) (string, error) {
	acc, ok := src.(metav1.Object)
	if !ok {
		return "", fmt.Errorf("source has to be a kubernetes metadata object")
	}

	srcKind := src.GetObjectKind().GroupVersionKind().Kind
	return srcKind + SourceDelimiter + acc.GetNamespace() + SourceDelimiter + acc.GetName(), nil
}

// ObjectFromDataObjectSource parses the source's kind, namespace and name from a src string.
func ObjectFromDataObjectSource(src string) (string, lsv1alpha1.ObjectReference, error) {
	splitValues := strings.Split(src, SourceDelimiter)
	if len(splitValues) != 3 {
		return "", lsv1alpha1.ObjectReference{}, fmt.Errorf("expected source definition with 3 paramters but got %d", len(splitValues))
	}

	kind, namespace, name := splitValues[0], splitValues[1], splitValues[2]
	return kind, lsv1alpha1.ObjectReference{Namespace: namespace, Name: name}, nil
}

// DataObjectSourceFromInstallation returns the data object source for a Installation.
func DataObjectSourceFromInstallation(src *lsv1alpha1.Installation) string {
	return fmt.Sprintf("Installation.%s.%s", src.GetNamespace(), src.GetName())
}

// DataObjectSourceFromExecution returns the data object source for a Execution.
func DataObjectSourceFromExecution(src *lsv1alpha1.Execution) string {
	return fmt.Sprintf("Execution.%s.%s", src.GetNamespace(), src.GetName())
}
