package common

import (
	"encoding/json"
	"fmt"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/versions/ocm.software/v3alpha1"
	v2 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/versions/v2"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const OCMSchemaVersion = "OCMSchemaVersion"

// DetermineOCMSchemaVersion analyzes the schema version against which should be templated.
// This can be defined as a OCMSchemaVersion blueprint annotation. If the blueprint does not have a respective
// annotation, the schema version is defaulted to the schema version of the provided component version.
func DetermineOCMSchemaVersion(blueprint *blueprints.Blueprint, componentVersion model.ComponentVersion) string {
	var ocmSchemaVersion string
	var ok bool

	if blueprint != nil {
		ocmSchemaVersion, ok = blueprint.Info.Annotations[OCMSchemaVersion]
		if ok {
			return ocmSchemaVersion
		}
	}
	if componentVersion != nil {
		ocmSchemaVersion = componentVersion.GetSchemaVersion()
	}

	return ocmSchemaVersion
}

func GetSchemaVersionFromCdMap(cdMap map[string]interface{}) (string, error) {
	var compdescSchemaVersion string
	apiVersion, ok := cdMap["apiVersion"]
	if ok {
		apiVersionCasted, ok := apiVersion.(string)
		if ok {
			compdescSchemaVersion = apiVersionCasted
			return compdescSchemaVersion, nil
		}
	}

	meta, ok := cdMap["meta"]
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
			} else {
				configuredSchemaVersion, ok := metaCasted["configuredSchemaVersion"]
				if ok {
					configuredSchemaVersionCasted, ok := configuredSchemaVersion.(string)
					if ok {
						compdescSchemaVersion = configuredSchemaVersionCasted
						return compdescSchemaVersion, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("Unable to determine component descriptor schema version.")
}

func ConvertCdMapToCompDescV2(inCdMap map[string]interface{}) (*types.ComponentDescriptor, error) {
	descriptor := types.ComponentDescriptor{}

	ocmSchemaVersion, err := GetSchemaVersionFromCdMap(inCdMap)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(inCdMap)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("invalid component descriptor: %s", err.Error()))
	}

	switch ocmSchemaVersion {
	case v3alpha1.GroupVersion:
		internalCd, err := compdesc.Decode(data)
		if err != nil {
			return nil, err
		}
		outCdData, err := compdesc.Encode(internalCd, compdesc.SchemaVersion(v2.SchemaVersion))
		if err != nil {
			return nil, err
		}
		err = runtime.DefaultYAMLEncoding.Unmarshal(outCdData, &descriptor)
		if err != nil {
			return nil, err
		}
	case v2.SchemaVersion:
		if err := codec.Decode(data, &descriptor); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown schema version")
	}

	return &descriptor, nil
}

func ConvertCompDescV2ToCdMap(cd cdv2.ComponentDescriptor, ocmSchemaVersion string) (map[string]interface{}, error) {
	data, err := json.Marshal(cd)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("invalid component descriptor: %s", err.Error()))
	}
	switch ocmSchemaVersion {
	case v3alpha1.GroupVersion:
		internalCd, err := compdesc.Decode(data)
		if err != nil {
			return nil, err
		}
		data, err = compdesc.Encode(internalCd, compdesc.SchemaVersion(v2.SchemaVersion))
		if err != nil {
			return nil, err
		}
	case v2.SchemaVersion:
	default:
		return nil, fmt.Errorf("unknown schema version")
	}

	cdMap := map[string]interface{}{}
	err = runtime.DefaultYAMLEncoding.Unmarshal(data, &cdMap)
	if err != nil {
		return nil, err
	}
	return cdMap, nil
}
