// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gotemplate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	gotmpl "text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/vfs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/landscaper/targetresolver"
	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	lstmpl "github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/common"
	"github.com/gardener/landscaper/pkg/utils/clusters"
)

// LandscaperSprigFuncMap returns the sanitized spring function map.
func LandscaperSprigFuncMap() gotmpl.FuncMap {
	fm := sprig.FuncMap()
	delete(fm, "env")
	delete(fm, "expandenv")
	return gotmpl.FuncMap(fm)
}

// LandscaperTplFuncMap contains all additional landscaper functions that are
// available in the executors templates.
func LandscaperTplFuncMap(blueprint *blueprints.Blueprint,
	componentVersion model.ComponentVersion,
	componentVersions *model.ComponentVersionList,
	targetResolver targetresolver.TargetResolver) (map[string]interface{}, error) {

	ocmSchemaVersion := common.DetermineOCMSchemaVersion(blueprint, componentVersion)

	cd, err := model.GetComponentDescriptor(componentVersion)
	if err != nil {
		return nil, fmt.Errorf("unable to get component descriptor to register go template functions: %w", err)
	}

	cdList, err := model.ConvertComponentVersionList(componentVersions)
	if err != nil {
		return nil, fmt.Errorf("unable to convert component descriptor list to register go template functions: %w", err)
	}

	funcs := map[string]interface{}{
		"readFile": readFileFunc(blueprint.Fs),
		"readDir":  readDir(blueprint.Fs),

		"toYaml": toYAML,

		"parseOCIRef":   lstmpl.ParseOCIReference,
		"ociRefRepo":    getOCIReferenceRepository,
		"ociRefVersion": getOCIReferenceVersion,
		"resolve":       resolveArtifactFunc(componentVersion),

		"getResource":          getResourceGoFunc(cd),
		"getResources":         getResourcesGoFunc(cd),
		"getComponent":         getComponentGoFunc(cd, cdList, ocmSchemaVersion),
		"getRepositoryContext": getEffectiveRepositoryContextGoFunc,

		"getShootAdminKubeconfig":                            getShootAdminKubeconfigGoFunc(targetResolver),
		"getShootAdminKubeconfigWithExpirationTimestamp":     getShootAdminKubeconfigWithExpirationTimestampGoFunc(targetResolver),
		"getServiceAccountKubeconfig":                        getServiceAccountKubeconfigGoFunc(targetResolver),
		"getServiceAccountKubeconfigWithExpirationTimestamp": getServiceAccountKubeconfigWithExpirationTimestampGoFunc(targetResolver),
		"getOidcKubeconfig":                                  getOidcKubeconfigGoFunc(targetResolver),
	}

	return funcs, nil
}

// readFileFunc returns a function that reads a file from a location in a filesystem
func readFileFunc(fs vfs.FileSystem) func(path string) []byte {
	return func(path string) []byte {
		file, err := vfs.ReadFile(fs, path)
		if err != nil {
			// maybe we should ignore the error and return an empty byte array
			panic(err)
		}
		return file
	}
}

// readDir lists all files of directory
func readDir(fs vfs.FileSystem) func(path string) []os.FileInfo {
	return func(path string) []os.FileInfo {
		files, err := vfs.ReadDir(fs, path)
		if err != nil {
			// maybe we should ignore the error and return an empty byte array
			panic(err)
		}
		return files
	}
}

// toYAML takes an interface, marshals it to yaml, and returns a string. It will
// always return a string, even on marshal error (empty string).
//
// This is designed to be called from a template.
func toYAML(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}

// getOCIReferenceVersion returns the version of a oci reference
func getOCIReferenceVersion(ref string) string {
	return lstmpl.ParseOCIReference(ref)[1]
}

// getOCIReferenceRepository returns the repository of a oci reference
func getOCIReferenceRepository(ref string) string {
	return lstmpl.ParseOCIReference(ref)[0]
}

// resolveArtifactFunc returns a function that can resolve artifact defined by a component descriptor access
func resolveArtifactFunc(componentVersion model.ComponentVersion) func(access map[string]interface{}) ([]byte, error) {
	return func(access map[string]interface{}) ([]byte, error) {
		ctx := context.Background()
		defer ctx.Done()

		cv, ok := componentVersion.(*cnudie.ComponentVersion)
		if !ok {
			return nil, errors.New("this functionality has been deprecated for usage with the new library ")
		}

		if componentVersion == nil {
			return nil, fmt.Errorf("unable to resolve artifact, because no component version is provided")
		}

		blobResolver, err := cv.GetBlobResolver()
		if err != nil {
			return nil, fmt.Errorf("unable to get blob resolver to resolve artifact: %w", err)
		}

		var data bytes.Buffer
		if _, err := blobResolver.Resolve(ctx, types.Resource{Access: cdv2.NewUnstructuredType(access["type"].(string), access)}, &data); err != nil {
			panic(err)
		}
		return data.Bytes(), nil
	}
}

func getResourcesGoFunc(cd *types.ComponentDescriptor) func(...interface{}) []map[string]interface{} {
	return func(args ...interface{}) []map[string]interface{} {
		if cd == nil {
			panic("Unable to search for a resource as no ComponentDescriptor is defined.")
		}
		resources, err := lstmpl.ResolveResources(cd, args)
		if err != nil {
			panic(err)
		}

		data, err := json.Marshal(resources)
		if err != nil {
			panic(err)
		}

		parsedResources := []map[string]interface{}{}
		if err := json.Unmarshal(data, &parsedResources); err != nil {
			panic(err)
		}
		return parsedResources
	}
}

func getResourceGoFunc(cd *types.ComponentDescriptor) func(args ...interface{}) map[string]interface{} {
	return func(args ...interface{}) map[string]interface{} {
		if cd == nil {
			panic("Unable to search for a resource as no ComponentDescriptor is defined.")
		}
		resources, err := lstmpl.ResolveResources(cd, args)
		if err != nil {
			panic(err)
		}

		// resources must be at least one, otherwise an error will be thrown
		data, err := json.Marshal(resources[0])
		if err != nil {
			panic(err)
		}

		parsedResource := map[string]interface{}{}
		if err := json.Unmarshal(data, &parsedResource); err != nil {
			panic(err)
		}
		return parsedResource
	}
}

func getEffectiveRepositoryContextGoFunc(arg interface{}) map[string]interface{} {
	if arg == nil {
		panic("Unable to get effective component descriptor as no ComponentDescriptor is defined.")
	}

	cdMap, ok := arg.(map[string]interface{})
	if !ok {
		panic("invalid component descriptor")
	}
	cd, err := common.ConvertMapCdToCompDescV2(cdMap)
	if err != nil {
		return nil
	}

	data, err := json.Marshal(cd.GetEffectiveRepositoryContext())
	if err != nil {
		panic(fmt.Sprintf("unable to serialize repository context: %s", err.Error()))
	}

	parsedRepoCtx := map[string]interface{}{}
	if err := json.Unmarshal(data, &parsedRepoCtx); err != nil {
		panic(fmt.Sprintf("unable to deserialize repository context: %s", err.Error()))
	}
	return parsedRepoCtx
}

func getComponentGoFunc(cd *types.ComponentDescriptor, list *types.ComponentDescriptorList, schemaVersion string) func(args ...interface{}) map[string]interface{} {
	return func(args ...interface{}) map[string]interface{} {
		if cd == nil {
			panic("Unable to search for a component as no ComponentDescriptor is defined.")
		}
		components, err := lstmpl.ResolveComponents(cd, list, schemaVersion, args)
		if err != nil {
			panic(err)
		}

		// resources must be at least one, otherwise an error will be thrown
		data, err := json.Marshal(components[0])
		if err != nil {
			panic(err)
		}

		parsedComponent := map[string]interface{}{}
		if err := json.Unmarshal(data, &parsedComponent); err != nil {
			panic(err)
		}
		return parsedComponent
	}
}

func getShootAdminKubeconfigGoFunc(targetResolver targetresolver.TargetResolver) func(args ...interface{}) (string, error) {
	return func(args ...interface{}) (string, error) {
		res, err := getShootAdminKubeconfigGoFunc_helper(targetResolver, false, args...)
		if err != nil {
			return "", err
		}
		return res.(string), nil
	}
}

func getShootAdminKubeconfigWithExpirationTimestampGoFunc(targetResolver targetresolver.TargetResolver) func(args ...interface{}) (map[string]interface{}, error) {
	return func(args ...interface{}) (map[string]interface{}, error) {
		res, err := getShootAdminKubeconfigGoFunc_helper(targetResolver, true, args...)
		if err != nil {
			return nil, err
		}
		return res.(map[string]interface{}), nil
	}
}

func getShootAdminKubeconfigGoFunc_helper(targetResolver targetresolver.TargetResolver, includeExpirationTimestamp bool, args ...interface{}) (interface{}, error) {
	if len(args) != 4 {
		return "", fmt.Errorf("templating function getShootAdminKubeconfig expects 4 arguments: shoot name, shoot namespace, expiration seconds, and target for garden project ")
	}

	shootName, ok := args[0].(string)
	if !ok {
		return "", fmt.Errorf("templating function getShootAdminKubeconfig expects a string as 1st argument, namely the shoot name")
	}

	shootNamespace, ok := args[1].(string)
	if !ok {
		return "", fmt.Errorf("templating function getShootAdminKubeconfig expects a string as 2nd argument, namely the shoot namespace")
	}

	expirationSeconds, err := toInt64(args[2])
	if err != nil {
		return "", fmt.Errorf("templating function getShootAdminKubeconfig expects an integer as 3rd argument, namely the expiration seconds: %w", err)
	}

	targetObj := args[3]
	targetBytes, err := json.Marshal(targetObj)
	if err != nil {
		return "", fmt.Errorf("templating function getShootAdminKubeconfig expects a target object as 4th argument: error during marshaling: %w", err)
	}

	target := &v1alpha1.Target{}
	err = json.Unmarshal(targetBytes, target)
	if err != nil {
		return "", fmt.Errorf("templating function getShootAdminKubeconfig expects a target object as 4th argument: error during unmarshaling: %w", err)
	}

	ctx := context.Background()
	shootClient, err := clusters.NewShootClientFromTarget(ctx, target, targetResolver)
	if err != nil {
		return "", err
	}

	kcfg, expirationTimestamp, err := shootClient.GetShootAdminKubeconfig(ctx, shootName, shootNamespace, expirationSeconds)
	if err != nil {
		return "", err
	}

	if includeExpirationTimestamp {
		return kubeconfigWithExpirationTimestamp(kcfg, expirationTimestamp), nil
	}
	return kcfg, nil
}

func getServiceAccountKubeconfigGoFunc(targetResolver targetresolver.TargetResolver) func(args ...interface{}) (string, error) {
	return func(args ...interface{}) (string, error) {
		res, err := getServiceAccountKubeconfigGoFunc_helper(targetResolver, false, args...)
		if err != nil {
			return "", err
		}
		return res.(string), nil
	}
}

func getServiceAccountKubeconfigWithExpirationTimestampGoFunc(targetResolver targetresolver.TargetResolver) func(args ...interface{}) (map[string]interface{}, error) {
	return func(args ...interface{}) (map[string]interface{}, error) {
		res, err := getServiceAccountKubeconfigGoFunc_helper(targetResolver, true, args...)
		if err != nil {
			return nil, err
		}
		return res.(map[string]interface{}), nil
	}
}

func getServiceAccountKubeconfigGoFunc_helper(targetResolver targetresolver.TargetResolver, includeExpirationTimestamp bool, args ...interface{}) (interface{}, error) {
	if len(args) != 4 {
		return "", fmt.Errorf("templating function getServiceAccountToken expects 4 arguments: service account name, service account namespace, expiration seconds, and target")
	}

	serviceAccountName, ok := args[0].(string)
	if !ok {
		return "", fmt.Errorf("templating function getServiceAccountToken expects a string as 1st argument, namely the service account name")
	}

	serviceAccountNamespace, ok := args[1].(string)
	if !ok {
		return "", fmt.Errorf("templating function getServiceAccountToken expects a string as 2nd argument, namely the service account namespace")
	}

	expirationSeconds, err := toInt64(args[2])
	if err != nil {
		return "", fmt.Errorf("templating function getServiceAccountToken expects an integer as 3rd argument, namely the expiration seconds: %w", err)
	}

	targetObj := args[3]
	targetBytes, err := json.Marshal(targetObj)
	if err != nil {
		return "", fmt.Errorf("templating function getServiceAccountToken expects a target object as 4th argument: error during marshaling: %w", err)
	}

	target := &v1alpha1.Target{}
	err = json.Unmarshal(targetBytes, target)
	if err != nil {
		return "", fmt.Errorf("templating function getServiceAccountToken expects a target object as 4th argument: error during unmarshaling: %w", err)
	}

	ctx := context.Background()
	tokenClient, err := clusters.NewTokenClientFromTarget(ctx, target, targetResolver)
	if err != nil {
		return "", err
	}

	kcfg, expirationTimestamp, err := tokenClient.GetServiceAccountKubeconfig(ctx, serviceAccountName, serviceAccountNamespace, expirationSeconds)
	if err != nil {
		return "", err
	}

	if includeExpirationTimestamp {
		return kubeconfigWithExpirationTimestamp(kcfg, expirationTimestamp), nil
	}
	return kcfg, nil
}

func getOidcKubeconfigGoFunc(targetResolver targetresolver.TargetResolver) func(args ...interface{}) (string, error) {
	return func(args ...interface{}) (string, error) {
		if len(args) != 3 {
			return "", fmt.Errorf("templating function getOidcKubeconfig expects 3 arguments: issuer url, client id, and target")
		}

		issuerURL, ok := args[0].(string)
		if !ok {
			return "", fmt.Errorf("templating function getOidcKubeconfig expects a string as 1st argument, namely the issuer url")
		}

		clientID, ok := args[1].(string)
		if !ok {
			return "", fmt.Errorf("templating function getOidcKubeconfig expects a string as 2nd argument, namely the client id")
		}

		targetObj := args[2]
		targetBytes, err := json.Marshal(targetObj)
		if err != nil {
			return "", fmt.Errorf("templating function getOidcKubeconfig expects a target object as 3rd argument: error during marshaling: %w", err)
		}

		target := &v1alpha1.Target{}
		err = json.Unmarshal(targetBytes, target)
		if err != nil {
			return "", fmt.Errorf("templating function getOidcKubeconfig expects a target object as 3rd argument: error during unmarshaling: %w", err)
		}

		ctx := context.Background()
		return clusters.BuildOIDCKubeconfig(ctx, issuerURL, clientID, target, targetResolver)
	}
}

func toInt64(value interface{}) (int64, error) {
	switch n := value.(type) {
	case int64:
		return n, nil
	case int32:
		return int64(n), nil
	case int16:
		return int64(n), nil
	case int8:
		return int64(n), nil
	case int:
		return int64(n), nil
	case float64:
		return int64(n), nil
	case float32:
		return int64(n), nil
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func kubeconfigWithExpirationTimestamp(kcfg string, expirationTimestamp metav1.Time) map[string]interface{} {
	return map[string]interface{}{
		"kubeconfig":                  kcfg,
		"expirationTimestamp":         expirationTimestamp.Unix(),
		"expirationTimestampReadable": expirationTimestamp.Format(time.RFC3339),
	}
}
