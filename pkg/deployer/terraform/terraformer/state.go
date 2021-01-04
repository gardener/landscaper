// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraformer

import (
	"context"
	"encoding/json"
	"errors"

	corev1 "k8s.io/api/core/v1"

	kutils "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// GetOutputFromState get the outputs from the state.
func (t *Terraformer) GetOutputFromState(ctx context.Context) (json.RawMessage, error) {
	tfstate, err := t.getState(ctx)
	if err != nil {
		return nil, err
	}
	if len(tfstate) == 0 {
		return nil, errors.New("unable to extract output, state is empty")
	}
	return extractOutput(ctx, tfstate)
}

// extractOutput extracts the terraform outputs from a given state.
func extractOutput(ctx context.Context, tfstate []byte) (json.RawMessage, error) {
	var state map[string]interface{}
	if err := json.Unmarshal(tfstate, &state); err != nil {
		return nil, err
	}

	outputs, ok := state[TerraformStateOutputsKey]
	if !ok {
		return nil, errors.New("no outputs found in the terraform state")
	}
	return json.Marshal(outputs)
}

// getState returns the state as byte slice from the state ConfigMap.
func (t *Terraformer) getState(ctx context.Context) ([]byte, error) {
	configMap := &corev1.ConfigMap{}
	if err := t.kubeClient.Get(ctx, kutils.ObjectKey(t.StateConfigMapName, t.Namespace), configMap); err != nil {
		return nil, err
	}
	return []byte(configMap.Data[TerraformStateKey]), nil
}
