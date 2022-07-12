package imports

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
)

type importsHashes struct {
	DataObjects              map[string]string
	Targets                  map[string]string
	TargetLists              map[string]string
	ComponentDescriptors     map[string]string
	ComponentDescriptorLists map[string]string
}

func ComputeImportsHash(imps *Imports) (string, error) {
	impsHashes := importsHashes{}

	impsHashes.DataObjects = make(map[string]string, len(imps.DataObjects))
	for k, v := range imps.DataObjects {
		impsHashes.DataObjects[k] = v.ComputeConfigGeneration()
	}

	impsHashes.Targets = make(map[string]string, len(imps.Targets))
	for k, v := range imps.Targets {
		impsHashes.DataObjects[k] = v.ComputeConfigGeneration()
	}

	impsHashes.TargetLists = make(map[string]string, len(imps.TargetLists))
	for k, v := range imps.TargetLists {
		impsHashes.DataObjects[k] = v.ComputeConfigGeneration()
	}

	impsHashes.ComponentDescriptors = make(map[string]string, len(imps.ComponentDescriptors))
	for k, v := range imps.ComponentDescriptors {
		impsHashes.DataObjects[k] = v.ComputeConfigGeneration()
	}

	impsHashes.ComponentDescriptorLists = make(map[string]string, len(imps.ComponentDescriptorLists))
	for k, v := range imps.ComponentDescriptorLists {
		impsHashes.DataObjects[k] = v.ComputeConfigGeneration()
	}

	impsHashesJson, err := json.Marshal(impsHashes)
	if err != nil {
		return "", err
	}

	h := sha1.New()
	_, err = h.Write(impsHashesJson)
	if err != nil {
		return "", err
	}
	result := hex.EncodeToString(h.Sum(nil))
	return result, nil
}
