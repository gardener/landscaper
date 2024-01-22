// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package spiff

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gardener/landscaper/pkg/components/ocmlib"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/utils"
	"github.com/open-component-model/ocm/pkg/mime"
	"github.com/open-component-model/ocm/pkg/runtime"
	"time"

	"github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/spiffing"
	spiffyaml "github.com/mandelsoft/spiff/yaml"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/versions/ocm.software/v3alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/landscaper/targetresolver"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/common"
	"github.com/gardener/landscaper/pkg/utils/clusters"
)

func LandscaperSpiffFuncs(blueprint *blueprints.Blueprint, functions spiffing.Functions, componentVersion model.ComponentVersion, componentVersions *model.ComponentVersionList, targetResolver targetresolver.TargetResolver) error {
	ocmSchemaVersion := common.DetermineOCMSchemaVersion(blueprint, componentVersion)

	cd, err := model.GetComponentDescriptor(componentVersion)
	if err != nil {
		return fmt.Errorf("unable to get component descriptor to register spiff functions: %w", err)
	}

	cdList, err := model.ConvertComponentVersionList(componentVersions)
	if err != nil {
		return fmt.Errorf("unable to convert component descriptor list to register spiff functions: %w", err)
	}

	functions.RegisterFunction("getResource", spiffResolveResources(cd))
	functions.RegisterFunction("getResourceKey", spiffGetResourceKey(componentVersion))
	functions.RegisterFunction("getResourceContent", spiffGetResourceContent(componentVersion))
	functions.RegisterFunction("getComponent", spiffResolveComponent(cd, cdList, ocmSchemaVersion))
	functions.RegisterFunction("parseOCIRef", parseOCIReference)
	functions.RegisterFunction("ociRefRepo", getOCIReferenceRepository)
	functions.RegisterFunction("ociRefVersion", getOCIReferenceVersion)
	functions.RegisterFunction("getShootAdminKubeconfig", getShootAdminKubeconfigSpiffFunc(targetResolver, false))
	functions.RegisterFunction("getShootAdminKubeconfigWithExpirationTimestamp", getShootAdminKubeconfigSpiffFunc(targetResolver, true))
	functions.RegisterFunction("getServiceAccountKubeconfig", getServiceAccountKubeconfigSpiffFunc(targetResolver, false))
	functions.RegisterFunction("getServiceAccountKubeconfigWithExpirationTimestamp", getServiceAccountKubeconfigSpiffFunc(targetResolver, true))
	functions.RegisterFunction("getOidcKubeconfig", getOidcKubeconfigSpiffFunc(targetResolver))

	return nil
}

func spiffResolveResources(cd *types.ComponentDescriptor) func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	return func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
		info := dynaml.DefaultInfo()

		// if the first input argument is a component descriptor in schema version v3alpha1, convert it to
		// schema version v2 for internal processing
		var err error
		compdescSchemaVersion := ""
		inCdMap, ok := arguments[0].(map[string]interface{})
		if ok {
			compdescSchemaVersion, err = common.GetSchemaVersionFromMapCd(inCdMap)
			if err != nil {
				panic(err)
			}
			if compdescSchemaVersion == v3alpha1.GroupVersion {
				arguments[0], err = common.ConvertMapCdToCompDescV2(inCdMap)
				if err != nil {
					panic("Unable to convert component descriptor to internal schema version")
				}
			}
		}

		data, err := spiffyaml.Marshal(spiffyaml.NewNode(arguments, ""))
		if err != nil {
			return info.Error(err.Error())
		}
		var val []interface{}
		if err := yaml.Unmarshal(data, &val); err != nil {
			return info.Error(err.Error())
		}

		resources, err := template.ResolveResources(cd, val)
		if err != nil {
			return info.Error(err.Error())
		}

		// resources must be at least one, otherwise an error will be thrown
		data, err = json.Marshal(resources[0])
		if err != nil {
			return info.Error(err.Error())
		}

		node, err := spiffyaml.Parse("", data)
		if err != nil {
			return info.Error(err.Error())
		}
		result, err := binding.Flow(node, false)
		if err != nil {
			return info.Error(err.Error())
		}

		return result.Value(), info, true
	}
}

// getResourceKeyGoFunc returns a function that resolves a relative resource reference
// (https://github.com/open-component-model/ocm-spec/blob/restruc3/doc/05-guidelines/03-references.md#relative-artifact-references).
// Its input is either a relative artifact reference in the format described by ocm or a file-path like expression
// like cd://componentReferences/referenceName1/componentReferences/referenceName2/resources/resourceName1 as defined
// here pkg/landscaper/registry/components/cdutils/uri.go.
// Based on an ocm component version given as input parameter, it constructs the global identity of this resource
// (https://github.com/open-component-model/ocm-spec/blob/restruc3/doc/01-model/03-elements-sub.md#identifiers) and
// returns a base64 encoded string representation of that global identity. This base64 encoded string acts as key
// which can be used by the deployers to fetch the resource content.
func spiffGetResourceKey(cv model.ComponentVersion) func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	return func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
		info := dynaml.DefaultInfo()

		if arguments == nil {
			return info.Error("unable to provide key for empty relative artifact reference")
		}
		if len(arguments) > 1 {
			return info.Error(fmt.Sprintf("this function requires 1 argument, but %v were provided", len(arguments)))
		}

		ocmlibCv, ok := cv.(*ocmlib.ComponentVersion)
		if !ok {
			return info.Error("unable to use this function without useOCM set to true")
		}
		compvers := ocmlibCv.GetOCMObject()

		var resourceRefStr string
		if ref, ok := arguments[0].(string); ok {
			resourceRefStr = ref
		} else if resourceRefData, err := spiffyaml.ValueToJSON(arguments[0]); err == nil {
			resourceRefStr = string(resourceRefData)
		} else {
			return info.Error("unable to parse input")
		}

		resourceRef, err := common.ParseResourceReference(resourceRefStr)
		if err != nil {
			return info.Error(err)
		}

		// if we ever migrate to an approach where a webserver fetches the resources for the deployers, instead
		// of passing the global id to the deployers, we should merely parse the relative reference to the deployer,
		// which the deployer would forward to the webserver and the webserver determines the root component to resolve
		// this reference by watching the installation (this way, we would ensure that the deployer can only get
		// resources from its legitimate component)
		resource, resourceCv, err := utils.ResolveResourceReference(compvers, *resourceRef, nil)
		if err != nil {
			return info.Error("unable to resolve relative resource reference: %w", err)
		}

		globalId := model.GlobalResourceIdentity{
			ComponentIdentity: model.ComponentIdentity{
				Name:    resourceCv.GetName(),
				Version: resourceCv.GetVersion(),
			},
			ResourceIdentity: resource.Meta().GetIdentity(resourceCv.GetDescriptor().Resources),
		}

		globalIdData, err := runtime.DefaultYAMLEncoding.Marshal(globalId)
		if err != nil {
			return info.Error("unable to marshal global resource identity: %w", err)
		}

		key := base64.StdEncoding.EncodeToString(globalIdData)

		return key, info, true
	}
}

// getResourceContentGoFunc returns a function that resolves a relative resource reference
// (https://github.com/open-component-model/ocm-spec/blob/restruc3/doc/05-guidelines/03-references.md#relative-artifact-references),
// based on an ocm component version given as input parameter and returns the content of the corresponding resource.
func spiffGetResourceContent(cv model.ComponentVersion) func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	return func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
		info := dynaml.DefaultInfo()

		if arguments == nil {
			return info.Error("unable to provide key for empty relative artifact reference")
		}
		if len(arguments) > 1 {
			return info.Error("this function only requires a single argument, but multiple were provided")
		}

		ocmlibCv, ok := cv.(*ocmlib.ComponentVersion)
		if !ok {
			return info.Error("unable to use this function without useOCM set to true")
		}
		compvers := ocmlibCv.GetOCMObject()

		var resourceRefStr string
		if ref, ok := arguments[0].(string); ok {
			resourceRefStr = ref
		} else if resourceRefData, err := spiffyaml.ValueToJSON(arguments[0]); err == nil {
			resourceRefStr = string(resourceRefData)
		} else {
			return info.Error("unable to parse input")
		}

		resourceRef, err := common.ParseResourceReference(resourceRefStr)
		if err != nil {
			return info.Error(err)
		}

		resource, _, err := utils.ResolveResourceReference(compvers, *resourceRef, nil)
		if err != nil {
			return info.Error("unable to resolve relative resource reference: %w", err)
		}

		m, err := resource.AccessMethod()
		if err != nil {
			return info.Error("unable to get access method for resource: %w", err)
		}

		data, err := m.Get()
		if err != nil {
			return info.Error("unable to read resource content: %w", err)
		}

		var out []byte
		switch m.MimeType() {
		case mime.MIME_OCTET:
			dst := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
			base64.StdEncoding.Encode(dst, data)
			out = dst
		default:
			out = data
		}

		node, err := spiffyaml.Unmarshal("data", out)
		if err != nil {
			return info.Error("unable to marshal resource into spiff node")
		}

		return node, info, true
	}
}

func spiffResolveComponent(cd *types.ComponentDescriptor, cdList *types.ComponentDescriptorList, ocmSchemaVersion string) func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	return func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
		info := dynaml.DefaultInfo()
		data, err := spiffyaml.Marshal(spiffyaml.NewNode(arguments, ""))
		if err != nil {
			return info.Error(err.Error())
		}
		var val []interface{}
		if err := yaml.Unmarshal(data, &val); err != nil {
			return info.Error(err.Error())
		}

		components, err := template.ResolveComponents(cd, cdList, ocmSchemaVersion, val)
		if err != nil {
			return info.Error(err.Error())
		}

		// resources must be at least one, otherwise an error will be thrown
		data, err = json.Marshal(components[0])
		if err != nil {
			return info.Error(err.Error())
		}

		node, err := spiffyaml.Parse("", data)
		if err != nil {
			return info.Error(err.Error())
		}
		result, err := binding.Flow(node, false)
		if err != nil {
			return info.Error(err.Error())
		}

		return result.Value(), info, true
	}
}

func parseOCIReference(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	info := dynaml.DefaultInfo()
	if len(arguments) > 1 {
		return info.Error("Too many arguments for parseOCIReference. Expected 1 reference.")
	}
	ref, ok := arguments[0].(string)
	if !ok {
		return info.Error("Invalid argument: string expected")
	}
	data, err := yaml.Marshal(template.ParseOCIReference(ref))
	if err != nil {
		return info.Error(err.Error())
	}

	node, err := spiffyaml.Parse("", data)
	if err != nil {
		return info.Error(err.Error())
	}

	result, err := binding.Flow(node, false)
	if err != nil {
		return info.Error(err.Error())
	}

	return result.Value(), info, true
}

func getOCIReferenceRepository(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	info := dynaml.DefaultInfo()
	if len(arguments) > 1 {
		return info.Error("Too many arguments for parseOCIReference. Expected 1 reference.")
	}
	ref := arguments[0].(string)
	data, err := yaml.Marshal(template.ParseOCIReference(ref)[0])
	if err != nil {
		return info.Error(err.Error())
	}

	node, err := spiffyaml.Parse("", data)
	if err != nil {
		return info.Error(err.Error())
	}

	result, err := binding.Flow(node, false)
	if err != nil {
		return info.Error(err.Error())
	}

	return result.Value(), info, true
}

func getOCIReferenceVersion(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	info := dynaml.DefaultInfo()
	if len(arguments) > 1 {
		return info.Error("Too many arguments for parseOCIReference. Expected 1 reference.")
	}
	ref := arguments[0].(string)
	data, err := yaml.Marshal(template.ParseOCIReference(ref)[1])
	if err != nil {
		return info.Error(err.Error())
	}

	node, err := spiffyaml.Parse("", data)
	if err != nil {
		return info.Error(err.Error())
	}

	result, err := binding.Flow(node, false)
	if err != nil {
		return info.Error(err.Error())
	}

	return result.Value(), info, true
}

func getShootAdminKubeconfigSpiffFunc(targetResolver targetresolver.TargetResolver, includeExpirationTimestamp bool) dynaml.Function {
	return func(args []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
		info := dynaml.DefaultInfo()
		if len(args) != 4 {
			return info.Error("templating function getShootAdminKubeconfig expects 4 arguments: shoot name, shoot namespace, expiration seconds, and target for garden project ")
		}

		shootName, ok := args[0].(string)
		if !ok {
			return info.Error("templating function getShootAdminKubeconfig expects a string as 1st argument, namely the shoot name")
		}

		shootNamespace, ok := args[1].(string)
		if !ok {
			return info.Error("templating function getShootAdminKubeconfig expects a string as 2nd argument, namely the shoot namespace")
		}

		expirationSeconds, err := toInt64(args[2])
		if err != nil {
			return info.Error("templating function getShootAdminKubeconfig expects an integer as 3rd argument, namely the expiration seconds: %w", err)
		}

		targetObj := args[3]
		targetBytes, err := spiffyaml.Marshal(spiffyaml.NewNode(targetObj, ""))
		if err != nil {
			return info.Error("templating function getShootAdminKubeconfig expects a target object as 4th argument: error during marshaling: %w", err)
		}

		target := &lsv1alpha1.Target{}
		err = yaml.Unmarshal(targetBytes, target)
		if err != nil {
			return info.Error("templating function getShootAdminKubeconfig expects a target object as 4th argument: error during unmarshaling: %w", err)
		}

		ctx := context.Background()
		shootClient, err := clusters.NewShootClientFromTarget(ctx, target, targetResolver)
		if err != nil {
			return info.Error(err)
		}

		kcfg, expirationTimestamp, err := shootClient.GetShootAdminKubeconfig(ctx, shootName, shootNamespace, expirationSeconds)
		if err != nil {
			return info.Error(err)
		}

		if includeExpirationTimestamp {
			return kubeconfigWithExpirationTimestamp(kcfg, expirationTimestamp, info, binding)
		}
		return kcfg, info, true
	}
}

func getServiceAccountKubeconfigSpiffFunc(targetResolver targetresolver.TargetResolver, includeExpirationTimestamp bool) dynaml.Function {
	return func(args []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
		info := dynaml.DefaultInfo()
		if len(args) != 4 {
			return info.Error("templating function getServiceAccountToken expects 4 arguments: service account name, service account namespace, expiration seconds, and target")
		}

		serviceAccountName, ok := args[0].(string)
		if !ok {
			return info.Error("templating function getServiceAccountToken expects a string as 1st argument, namely the service account name")
		}

		serviceAccountNamespace, ok := args[1].(string)
		if !ok {
			return info.Error("templating function getServiceAccountToken expects a string as 2nd argument, namely the service account namespace")
		}

		expirationSeconds, err := toInt64(args[2])
		if err != nil {
			return info.Error("templating function getServiceAccountToken expects an integer as 3rd argument, namely the expiration seconds: %w", err)
		}

		targetObj := args[3]
		targetBytes, err := spiffyaml.Marshal(spiffyaml.NewNode(targetObj, ""))
		if err != nil {
			return info.Error("templating function getServiceAccountToken expects a target object as 4th argument: error during marshaling: %w", err)
		}

		target := &lsv1alpha1.Target{}
		err = yaml.Unmarshal(targetBytes, target)
		if err != nil {
			return info.Error("templating function getServiceAccountToken expects a target object as 4th argument: error during unmarshaling: %w", err)
		}

		ctx := context.Background()
		tokenClient, err := clusters.NewTokenClientFromTarget(ctx, target, targetResolver)
		if err != nil {
			return info.Error(err)
		}

		kcfg, expirationTimestamp, err := tokenClient.GetServiceAccountKubeconfig(ctx, serviceAccountName, serviceAccountNamespace, expirationSeconds)
		if err != nil {
			return info.Error(err)
		}

		if includeExpirationTimestamp {
			return kubeconfigWithExpirationTimestamp(kcfg, expirationTimestamp, info, binding)
		}
		return kcfg, info, true
	}
}

func getOidcKubeconfigSpiffFunc(targetResolver targetresolver.TargetResolver) dynaml.Function {
	return func(args []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
		info := dynaml.DefaultInfo()
		if len(args) != 3 {
			return info.Error("templating function getOidcKubeconfig expects 3 arguments: issuer url, client id, and target")
		}

		issuerURL, ok := args[0].(string)
		if !ok {
			return info.Error("templating function getOidcKubeconfig expects a string as 1st argument, namely the issuer url")
		}

		clientID, ok := args[1].(string)
		if !ok {
			return info.Error("templating function getOidcKubeconfig expects a string as 2nd argument, namely the client id")
		}

		targetObj := args[2]
		targetBytes, err := spiffyaml.Marshal(spiffyaml.NewNode(targetObj, ""))
		if err != nil {
			return info.Error("templating function getOidcKubeconfig expects a target object as 3rd argument: error during marshaling: %w", err)
		}

		target := &lsv1alpha1.Target{}
		err = yaml.Unmarshal(targetBytes, target)
		if err != nil {
			return info.Error("templating function getOidcKubeconfig expects a target object as 3rd argument: error during unmarshaling: %w", err)
		}

		ctx := context.Background()
		kcfg, err := clusters.BuildOIDCKubeconfig(ctx, issuerURL, clientID, target, targetResolver)
		if err != nil {
			return info.Error(err)
		}

		return kcfg, info, true
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

func kubeconfigWithExpirationTimestamp(kcfg string, expirationTimestamp metav1.Time, info dynaml.EvaluationInfo, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	rawData := map[string]interface{}{
		"kubeconfig":                  kcfg,
		"expirationTimestamp":         expirationTimestamp.Unix(),
		"expirationTimestampReadable": expirationTimestamp.Format(time.RFC3339),
	}

	data, err := yaml.Marshal(rawData)
	if err != nil {
		return info.Error(err.Error())
	}

	node, err := spiffyaml.Parse("", data)
	if err != nil {
		return info.Error(err.Error())
	}

	result, err := binding.Flow(node, false)
	if err != nil {
		return info.Error(err.Error())
	}

	return result.Value(), info, true
}
