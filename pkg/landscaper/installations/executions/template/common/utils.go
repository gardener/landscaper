package common

import (
	"encoding/json"
	"fmt"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	"strings"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	v1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/versions/ocm.software/v3alpha1"
	v2 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/versions/v2"
	"github.com/open-component-model/ocm/pkg/runtime"

	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

const (
	OCM_SCHEMA_VERSION      = "ocmSchemaVersion"
	SCHEMA_VERSION_V2       = cdv2.SchemaVersion
	SCHEMA_VERSION_V3ALPHA1 = v3alpha1.GroupVersion
)

// DetermineOCMSchemaVersion analyzes against which ocm schema version should be templated.
// This can be defined as a OCM_SCHEMA_VERSION blueprint annotation. If the blueprint does not have a respective
// annotation, the schema version is defaulted to the schema version of the provided component version.
func DetermineOCMSchemaVersion(blueprint *blueprints.Blueprint, componentVersion model.ComponentVersion) string {
	ocmSchemaVersion := ""

	if blueprint != nil && blueprint.Info != nil && blueprint.Info.Annotations != nil {
		schemaVersion, ok := blueprint.Info.Annotations[OCM_SCHEMA_VERSION]
		if ok {
			ocmSchemaVersion = schemaVersion
		}
	}
	if ocmSchemaVersion == "" && componentVersion != nil {
		ocmSchemaVersion = componentVersion.GetSchemaVersion()
	}

	return ocmSchemaVersion
}

// GetSchemaVersionFromMapCd takes a component descriptor that was unmarshalled into a map[string]interface{} and tries
// to extract the value of the schema version property.
func GetSchemaVersionFromMapCd(mapCd map[string]interface{}) (string, error) {
	var compdescSchemaVersion string

	// This code tries to get the schema version from a component descriptor adhering to schema version v3alpha1.
	apiVersion, ok := mapCd["apiVersion"]
	if ok {
		apiVersionCasted, ok := apiVersion.(string)
		if ok {
			compdescSchemaVersion = apiVersionCasted
			return compdescSchemaVersion, nil
		}
	}

	// This code tries to get the schema version from a component descriptor adhering to schema version v2.
	meta, ok := mapCd["meta"]
	if ok {
		metaCasted, ok := meta.(map[string]interface{})
		if ok {
			schemaVersion, ok := metaCasted["schemaVersion"]
			if ok {
				schemaVersionCasted, ok := schemaVersion.(string)
				if ok {
					compdescSchemaVersion = schemaVersionCasted
					return compdescSchemaVersion, nil
				}
			}
		}
	}

	return "", fmt.Errorf("unable to determine component descriptor schema version")
}

// ConvertMapCdToCompDescV2 takes a component descriptor that was unmarshalled into a map[string]interface{} and
// converts it to a component descriptor struct resembling the legacy component-spec schema version v2. The function
// can deal with component descriptors adhering to this legacy v2, the ocm-spec v2 and the ocm-spec v3alpha1 schema
// version. The legacy v2 schema version is currently used as the internal component descriptor version of the
// landscaper for compatibility reasons. It is largely compatible to the ocm-spec schema version v2.
func ConvertMapCdToCompDescV2(mapCd map[string]interface{}) (*types.ComponentDescriptor, error) {
	descriptor := types.ComponentDescriptor{}

	ocmSchemaVersion, err := GetSchemaVersionFromMapCd(mapCd)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(mapCd)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("invalid component descriptor: %s", err.Error()))
	}

	switch ocmSchemaVersion {
	case SCHEMA_VERSION_V3ALPHA1:
		ocmlibCd, err := compdesc.Decode(data)
		if err != nil {
			return nil, err
		}
		cdData, err := compdesc.Encode(ocmlibCd, compdesc.SchemaVersion(v2.SchemaVersion))
		if err != nil {
			return nil, err
		}
		err = runtime.DefaultYAMLEncoding.Unmarshal(cdData, &descriptor)
		if err != nil {
			return nil, err
		}
	case SCHEMA_VERSION_V2:
		if err := codec.Decode(data, &descriptor); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown schema version")
	}

	return &descriptor, nil
}

// ConvertCompDescV2ToMapCd takes a component descriptor struct resembling the legacy component-spec schema version v2,
// converts it to the specified ocmSchemaVersion and unmarshals it into a map[string]interface{}. Possible
// ocmSchemaVersion values are currently v2 (which leads to a component descriptor adhering to the legacy component-spec
// v2 schema version for compatibility reasons) and v3alpha1.
func ConvertCompDescV2ToMapCd(cd cdv2.ComponentDescriptor, ocmSchemaVersion string) (map[string]interface{}, error) {
	data, err := json.Marshal(cd)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("invalid component descriptor: %s", err.Error()))
	}
	switch ocmSchemaVersion {
	case SCHEMA_VERSION_V3ALPHA1:
		ocmlibCd, err := compdesc.Decode(data)
		if err != nil {
			return nil, err
		}
		data, err = compdesc.Encode(ocmlibCd, compdesc.SchemaVersion(v2.SchemaVersion))
		if err != nil {
			return nil, err
		}
	case SCHEMA_VERSION_V2:
	default:
		return nil, fmt.Errorf("unknown schema version")
	}

	mapCd := map[string]interface{}{}
	err = runtime.DefaultYAMLEncoding.Unmarshal(data, &mapCd)
	if err != nil {
		return nil, err
	}
	return mapCd, nil
}

func ParseResourceReference(ref string) (*v1.ResourceReference, error) {
	resourceRef := &v1.ResourceReference{}
	if strings.HasPrefix(ref, "cd://") {
		// assume that the resource is specified through a path expression
		resourceRefUri, err := cdutils.ParseURI(ref)
		if err != nil {
			return nil, fmt.Errorf("failed to parse argument into URI: %w", err)
		}
		resourceRef, err = resourceRefUri.AsRelativeResourceReference()
	} else {
		// assume that the resource is specified through a relative artifact reference
		// (https://github.com/open-component-model/ocm-spec/blob/restruc3/doc/05-guidelines/03-references.md#relative-artifact-references)
		err := runtime.DefaultYAMLEncoding.Unmarshal([]byte(ref), resourceRef)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal argument into a relative resource reference: %w", err)
		}
	}
	return resourceRef, nil
}
