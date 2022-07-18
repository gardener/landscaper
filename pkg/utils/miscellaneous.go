// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

// MergeMaps takes two maps <a>, <b> and merges them. If <b> defines a value with a key
// already existing in the <a> map, the <a> value for that key will be overwritten.
func MergeMaps(a, b map[string]interface{}) map[string]interface{} {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	var values = map[string]interface{}{}

	for i, v := range b {
		existing, ok := a[i]
		values[i] = v

		switch elem := v.(type) {
		case map[string]interface{}:
			if ok {
				if extMap, ok := existing.(map[string]interface{}); ok {
					values[i] = MergeMaps(extMap, elem)
				}
			}
		default:
			values[i] = v
		}
	}

	for i, v := range a {
		if _, ok := values[i]; !ok {
			values[i] = v
		}
	}

	return values
}

// JSONSerializeToGenericObject converts a typed struct to an generic interface using json serialization.
func JSONSerializeToGenericObject(in interface{}) (interface{}, error) {
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var val interface{}
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, err
	}
	return val, nil
}

// StringIsOneOf checks whether in is one of s.
func StringIsOneOf(in string, s ...string) bool {
	for _, search := range s {
		if search == in {
			return true
		}
	}
	return false
}

// GetSizeOfDirectory returns the size of all files in a root directory.
func GetSizeOfDirectory(filesystem vfs.FileSystem, root string) (int64, error) {
	var size int64
	err := vfs.Walk(filesystem, root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		size = size + info.Size()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return size, nil
}

// CopyFS copies all files and directories of a filesystem to another.
func CopyFS(src, dst vfs.FileSystem, srcPath, dstPath string) error {
	return vfs.Walk(src, srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		dstFilePath := filepath.Join(dstPath, path)
		if info.IsDir() {
			if err := dst.MkdirAll(dstFilePath, info.Mode()); err != nil {
				return err
			}
			return nil
		}

		file, err := src.OpenFile(path, os.O_RDONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()

		dstFile, err := dst.Create(dstFilePath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, file); err != nil {
			return err
		}
		return nil
	})
}

// YAMLReadFromFile reads a file from a filesystem and decodes it into the given obj
// using YAMl/JSON decoder.
func YAMLReadFromFile(fs vfs.FileSystem, path string, obj interface{}) error {
	data, err := vfs.ReadFile(fs, path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, obj)
}

// CheckForDuplicateExports takes a current installation and a list of existing installations and returns an error,
// if the current installation declares an export which is already declared by one of the existing installations.
// The function will return after the first found conflicts, but it will return all conflicts with the same existing installation.
func CheckForDuplicateExports(current *lsv1alpha1.Installation, existing []lsv1alpha1.Installation) error {
	curDataExp, curTargetExp := extractExportNames(current)

	if len(curDataExp) == 0 && len(curTargetExp) == 0 {
		// current installation does not export anything, therefore conflicts are not possible
		return nil
	}

	for _, inst := range existing {
		if current.Name == inst.Name {
			// make sure we don't compare the installation with itself, as this will always lead to conflicts if something is exported
			continue
		}
		existDataExp, existTargetExp := extractExportNames(&inst)
		commonDataExp := curDataExp.Intersection(existDataExp)
		commonTargetExp := curTargetExp.Intersection(existTargetExp)

		if len(commonDataExp) != 0 || len(commonTargetExp) != 0 {
			return fmt.Errorf("installation '%s/%s' has conflicting exports with installation '%s/%s': data exports [%v], target exports [%v]", current.Namespace, current.Name, inst.Namespace, inst.Name, strings.Join(commonDataExp.List(), ", "), strings.Join(commonTargetExp.List(), ", "))
		}
	}

	return nil
}

// extractExportNames returns two sets containing the names of the data and target exports of the given installation, respectively.
func extractExportNames(inst *lsv1alpha1.Installation) (sets.String, sets.String) {
	dataExports, targetExports := sets.NewString(), sets.NewString()

	if inst == nil {
		return dataExports, targetExports
	}

	for _, exp := range inst.Spec.Exports.Data {
		dataExports.Insert(exp.DataRef)
	}
	for _, exp := range inst.Spec.Exports.Targets {
		targetExports.Insert(exp.Target)
	}
	for exp := range inst.Spec.ExportDataMappings {
		dataExports.Insert(exp)
	}

	return dataExports, targetExports
}

// SetExclusiveOwnerReference is a wrapper around controllerutil.SetOwnerReference
// The first return value will contain an error if the object contains already an owner reference of the same kind but pointing to a different owner.
// The second return value is meant for unexpected errors during the process.
func SetExclusiveOwnerReference(owner client.Object, obj client.Object) (error, error) {
	gvk, err := apiutil.GVKForObject(owner, api.LandscaperScheme)
	if err != nil {
		return nil, fmt.Errorf("unable to determine GroupVersionKind for object %s: %w", client.ObjectKeyFromObject(owner).String(), err)
	}
	for _, own := range obj.GetOwnerReferences() {
		if own.Kind == gvk.Kind && own.UID != owner.GetUID() {
			return fmt.Errorf("object '%s' is already owned by another object with kind '%s' (%s)", client.ObjectKeyFromObject(obj).String(), gvk.Kind, own.Name), nil
		}
	}
	return nil, controllerutil.SetOwnerReference(owner, obj, api.LandscaperScheme)
}
