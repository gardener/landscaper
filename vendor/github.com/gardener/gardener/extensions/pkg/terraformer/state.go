// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package terraformer

import (
	"context"
	"encoding/json"
	"fmt"

	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type terraformStateV3 struct {
	Modules []struct {
		Outputs map[string]outputState `json:"outputs"`
	} `json:"modules"`
}

type outputState struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type terraformStateV4 struct {
	Outputs map[string]outputState `json:"outputs"`
}

// GetState returns the Terraform state as byte slice.
func (t *terraformer) GetState() ([]byte, error) {
	ctx := context.TODO()
	configMap := &corev1.ConfigMap{}
	if err := t.client.Get(ctx, kutil.Key(t.namespace, t.stateName), configMap); err != nil {
		return nil, err
	}

	return []byte(configMap.Data[StateKey]), nil
}

// GetStateOutputVariables returns the given <variable> from the given Terraform <stateData>.
// In case the variable was not found, an error is returned.
func (t *terraformer) GetStateOutputVariables(variables ...string) (map[string]string, error) {
	var (
		output = make(map[string]string)

		wantedVariables = sets.NewString(variables...)
		foundVariables  = sets.NewString()
	)

	stateConfigMap, err := t.GetState()
	if err != nil {
		return nil, err
	}

	if len(stateConfigMap) == 0 {
		return nil, &variablesNotFoundError{wantedVariables.List()}
	}

	outputVariables, err := getOutputVariables(stateConfigMap)
	if err != nil {
		return nil, err
	}

	for _, variable := range variables {
		if outputVariable, ok := outputVariables[variable]; ok {
			output[variable] = fmt.Sprint(outputVariable.Value)
			foundVariables.Insert(variable)
		}
	}

	if wantedVariables.Len() != foundVariables.Len() {
		return nil, &variablesNotFoundError{wantedVariables.Difference(foundVariables).List()}
	}

	return output, nil
}

// IsStateEmpty returns true if the Terraform state is empty, and false otherwise.
func (t *terraformer) IsStateEmpty() bool {
	state, err := t.GetState()
	if err != nil {
		return apierrors.IsNotFound(err)
	}
	return len(state) == 0
}

type variablesNotFoundError struct {
	variables []string
}

// Error prints the error message of the variablesNotFound error.
func (e *variablesNotFoundError) Error() string {
	return fmt.Sprintf("could not find all requested variables: %+v", e.variables)
}

// IsVariablesNotFoundError returns true if the error indicates that not all variables have been found.
func IsVariablesNotFoundError(err error) bool {
	switch err.(type) {
	case *variablesNotFoundError:
		return true
	}
	return false
}

func getOutputVariables(stateConfigMap []byte) (map[string]outputState, error) {
	version, err := sniffJSONStateVersion(stateConfigMap)
	if err != nil {
		return nil, err
	}

	var outputVariables map[string]outputState
	switch {
	case version == 2 || version == 3:
		var state terraformStateV3
		if err := json.Unmarshal(stateConfigMap, &state); err != nil {
			return nil, err
		}

		outputVariables = state.Modules[0].Outputs
	case version == 4:
		var state terraformStateV4
		if err := json.Unmarshal(stateConfigMap, &state); err != nil {
			return nil, err
		}

		outputVariables = state.Outputs
	default:
		return nil, fmt.Errorf("the state file uses format version %d, which is not supported by Terraformer", version)
	}

	return outputVariables, nil
}

func sniffJSONStateVersion(stateConfigMap []byte) (uint64, error) {
	type VersionSniff struct {
		Version *uint64 `json:"version"`
	}
	var sniff VersionSniff
	if err := json.Unmarshal(stateConfigMap, &sniff); err != nil {
		return 0, fmt.Errorf("the state file could not be parsed as JSON: %v", err)
	}

	if sniff.Version == nil {
		return 0, fmt.Errorf("the state file does not have a \"version\" attribute, which is required to identify the format version")
	}

	return *sniff.Version, nil
}

// Initialize implements
func (f StateConfigMapInitializerFunc) Initialize(ctx context.Context, c client.Client, namespace, name string) error {
	return f(ctx, c, namespace, name)
}

// CreateState create terraform state config map and use empty state.
// It does not create or update state ConfigMap if already exists.
func CreateState(ctx context.Context, c client.Client, namespace, name string) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Data: map[string]string{
			StateKey: "",
		},
	}

	if err := c.Create(ctx, configMap); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// Initialize implements StateConfigMapInitializer
func (cus CreateOrUpdateState) Initialize(ctx context.Context, c client.Client, namespace, name string) error {
	if cus.State == nil {
		return fmt.Errorf("missing state when creating or updating terraform state ConfigMap %s/%s", namespace, name)
	}
	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}

	_, err := controllerutil.CreateOrUpdate(ctx, c, configMap, func() error {
		if configMap.Data == nil {
			configMap.Data = make(map[string]string)
		}
		configMap.Data[StateKey] = *cus.State
		return nil
	})

	return err
}
